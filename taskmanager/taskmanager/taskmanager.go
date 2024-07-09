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

	"github.com/robfig/cron"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const JobCreatedEventSignature = "JobCreated(uint32,string,string,string)"

type TaskManager struct {
	client        *ethclient.Client
	contractAddr  common.Address
	jobCreatedSig common.Hash
}

type Task struct {
	TaskID   uint32 `json:"taskID"`
	TaskType string `json:"taskType"`
}

type JobCreatedEvent struct {
	JobID          uint32 `json:"jobID"`
	JobType        string `json:"jobType"`
	JobDescription string `json:"jobDescription"`
	JobURL         string `json:"jobURL"`
	Tasks          []Task `json:"tasks"`
	Timeframe      uint32 `json:"timeframe"`
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
			tm.AllocateTasks(vLog)
		}
	}
}

func (tm *TaskManager) AllocateTasks(vLog types.Log) {
	// Decode the event log
	var jobCreatedEvent JobCreatedEvent
	data := vLog.Data
	if err := decodeEventData(data, &jobCreatedEvent); err != nil {
		log.Printf("Failed to decode event data: %v", err)
		return
	}

	log.Printf("Decoded JobCreated event: %+v\n", jobCreatedEvent)

	// Schedule tasks to send to operator
	err := tm.scheduleTasks(jobCreatedEvent)
	if err != nil {
		log.Printf("Failed to schedule tasks: %v", err)
	}
}

func decodeEventData(data []byte, event *JobCreatedEvent) error {
	if len(data) < 160 {
		return fmt.Errorf("invalid event data length")
	}

	// Decode the uint32 jobID
	jobID := new(big.Int).SetBytes(data[:32]).Uint64()
	event.JobID = uint32(jobID)

	// Decode the timeframe
	timeframe := new(big.Int).SetBytes(data[32:64]).Uint64()
	event.Timeframe = uint32(timeframe)

	// Decode the offsets
	offset0 := new(big.Int).SetBytes(data[64:96]).Uint64()
	offset1 := new(big.Int).SetBytes(data[96:128]).Uint64()
	offset2 := new(big.Int).SetBytes(data[128:160]).Uint64()

	// Decode jobType
	jobTypeLength := new(big.Int).SetBytes(data[offset0 : offset0+32]).Uint64()
	event.JobType = string(data[offset0+32 : offset0+32+jobTypeLength])

	// Decode jobDescription
	jobDescriptionLength := new(big.Int).SetBytes(data[offset1 : offset1+32]).Uint64()
	event.JobDescription = string(data[offset1+32 : offset1+32+jobDescriptionLength])

	// Decode jobURL
	jobURLLength := new(big.Int).SetBytes(data[offset2 : offset2+32]).Uint64()
	event.JobURL = string(data[offset2+32 : offset2+32+jobURLLength])

	// Decode tasks
	tasksOffset := new(big.Int).SetBytes(data[160:192]).Uint64()
	tasksLength := new(big.Int).SetBytes(data[tasksOffset : tasksOffset+32]).Uint64()
	tasksData := data[tasksOffset+32 : tasksOffset+32+tasksLength]

	// Parse tasks JSON
	if err := json.Unmarshal(tasksData, &event.Tasks); err != nil {
		return fmt.Errorf("failed to unmarshal tasks JSON: %v", err)
	}

	return nil
}

func (tm *TaskManager) scheduleTasks(job JobCreatedEvent) error {
	// Create a cron job to send tasks to operator at equal intervals
	c := cron.New()

	// Calculate the interval between tasks
	if len(job.Tasks) == 0 {
		return fmt.Errorf("no tasks to schedule")
	}

	interval := job.Timeframe / uint32(len(job.Tasks))
	if interval == 0 {
		return fmt.Errorf("timeframe too short for the number of tasks")
	}

	for i, task := range job.Tasks {
		delay := time.Duration(interval*uint32(i)) * time.Second
		cronSchedule := fmt.Sprintf("@every %ds", delay)

		err := c.AddFunc(cronSchedule, func() {
			err := sendTaskToOperator(task)
			if err != nil {
				log.Printf("Failed to send task to operator: %v", err)
			}
		})

		if err != nil {
			return fmt.Errorf("failed to add cron job: %v", err)
		}
	}

	c.Start()
	defer c.Stop()

	return nil
}

func sendTaskToOperator(task Task) error {
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
