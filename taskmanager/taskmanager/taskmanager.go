package taskmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/Keeper/contracts" // Replace with the actual path to your generated contract bindings
)

const JobCreatedEventSignature = "JobCreated(uint32,string,string,uint256)"

type TaskManager struct {
	client        *ethclient.Client
	contractAddr  common.Address
	jobCreatedSig common.Hash
}

type Task struct {
	JobID        uint32 `json:"jobID"`
	TaskID       uint32 `json:"taskID"`
	ChainID      uint   `json:"chainID"`
	ContractAddr string `json:"contractAddr"`
	TargetFunc   string `json:"targetFunc"`
}

type Job struct {
	JobID        uint32   `json:"jobID"`
	JobType      string   `json:"jobType"`
	ContractAddr string   `json:"contractAddr"`
	ChainID      uint     `json:"chainID"`
	TargetFunc   string   `json:"targetFunc"`
	Timeframe    uint32   `json:"timeframe"`
	Tasks        []string `json:"tasks"`
}

func NewTaskManager(clientURL string, contractAddr string) (*TaskManager, error) {
	client, err := ethclient.Dial(clientURL)
	if err != nil {
		return nil, err
	}
	jobCreatedSig := crypto.Keccak256Hash([]byte(JobCreatedEventSignature))
	return &TaskManager{
		client:        client,
		contractAddr:  common.HexToAddress(contractAddr),
		jobCreatedSig: jobCreatedSig,
	}, nil
}

func (tm *TaskManager) ListenForEvents() {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{tm.contractAddr},
		Topics:    [][]common.Hash{{tm.jobCreatedSig}},
	}

	logs := make(chan types.Log)
	ctx := context.Background()

	sub, err := tm.client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		log.Fatalf("Failed to subscribe to filter logs: %v", err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatalf("Subscription error: %v", err)
		case vLog := <-logs:
			log.Printf("Received JobCreated event log: %+v\n", vLog)
			tm.ProcessJob(vLog)
		}
	}
}

func (tm *TaskManager) ProcessJob(vLog types.Log) {
	var job Job
	err := tm.decodeJobCreatedEvent(vLog, &job)
	if err != nil {
		log.Printf("Failed to decode JobCreated event: %v", err)
		return
	}

	log.Printf("Decoded Job: %+v\n", job)

	err = tm.scheduleTasks(job)
	if err != nil {
		log.Printf("Failed to schedule tasks: %v", err)
	}
}

func (tm *TaskManager) decodeJobCreatedEvent(vLog types.Log, job *Job) error {
	if len(vLog.Data) < 128 {
		return fmt.Errorf("invalid event data length")
	}

	job.JobID = uint32(new(big.Int).SetBytes(vLog.Data[:32]).Uint64())
	job.JobType = string(vLog.Data[32:64])
	job.ContractAddr = common.BytesToAddress(vLog.Data[64:96]).Hex()
	job.ChainID = uint(new(big.Int).SetBytes(vLog.Data[96:128]).Uint64())

	// Fetch additional job details from the contract
	jobDetails, err := tm.fetchJobDetails(job.JobID)
	if err != nil {
		return err
	}

	job.TargetFunc = jobDetails.TargetFunc
	job.Timeframe = jobDetails.Timeframe
	job.Tasks = jobDetails.Tasks

	return nil
}

func (tm *TaskManager) fetchJobDetails(jobID uint32) (*Job, error) {
	// Implement contract call to fetch job details
	// This is a placeholder and needs to be implemented based on your contract
	return &Job{
		TargetFunc: "exampleFunction",
		Timeframe:  3600, // 1 hour in seconds
		Tasks:      []string{"task1", "task2", "task3"},
	}, nil
}

func (tm *TaskManager) scheduleTasks(job Job) error {
	c := cron.New()

	// Create an instance of the KeeperNetworkTaskManager contract
	taskManager, err := contracts.NewKeeperNetworkTaskManager(tm.contractAddr, tm.client)
	if err != nil {
		return fmt.Errorf("failed to instantiate KeeperNetworkTaskManager contract: %v", err)
	}

	numTasks := len(job.Tasks)
	if numTasks == 0 {
		return fmt.Errorf("no tasks defined for job %d", job.JobID)
	}

	interval := time.Duration(job.Timeframe/uint32(numTasks)) * time.Second

	for i, taskType := range job.Tasks {
		taskID := uint32(i + 1)
		delay := time.Duration(i) * interval
		
		task := Task{
			JobID:        job.JobID,
			TaskID:       taskID,
			ChainID:      job.ChainID,
			ContractAddr: job.ContractAddr,
			TargetFunc:   job.TargetFunc,
		}

		// Create the task in the smart contract
		tx, err := taskManager.CreateTask(nil, job.JobID, taskID, taskType, "Scheduled")
		if err != nil {
			return fmt.Errorf("failed to create task in smart contract: %v", err)
		}

		// Wait for the transaction to be mined
		_, err = bind.WaitMined(context.Background(), tm.client, tx)
		if err != nil {
			return fmt.Errorf("failed to wait for CreateTask transaction to be mined: %v", err)
		}

		// Schedule the task execution
		_, err = c.AddFunc(fmt.Sprintf("@every %v", delay), func() {
			err := sendTaskToKeeper(task)
			if err != nil {
				log.Printf("Failed to send task to keeper: %v", err)
			}
		})

		if err != nil {
			return fmt.Errorf("failed to add cron job: %v", err)
		}
	}

	c.Start()
	return nil
}

func sendTaskToKeeper(task Task) error {
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}

	resp, err := http.Post("http://localhost:8081/executeTask", "application/json", strings.NewReader(string(taskJSON)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send task: %s", resp.Status)
	}

	return nil
}