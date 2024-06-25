package taskmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"

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

type JobCreatedEvent struct {
	JobID          uint32 `json:"jobID"`
	JobType        string `json:"jobType"`
	JobDescription string `json:"jobDescription"`
	JobURL         string `json:"jobURL"`
	Timeframe      uint32 `json:"timeframe"` // Add this line
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
			tm.AllocateTask(vLog)
		}
	}
}

func (tm *TaskManager) AllocateTask(vLog types.Log) {
	// Decode the event log
	var jobCreatedEvent JobCreatedEvent
	data := vLog.Data
	if err := decodeEventData(data, &jobCreatedEvent); err != nil {
		log.Printf("Failed to decode event data: %v", err)
		return
	}

	log.Printf("Decoded JobCreated event: %+v\n", jobCreatedEvent)

	// Schedule task to send to operator
	err := tm.scheduleTask(jobCreatedEvent)
	if err != nil {
		log.Printf("Failed to schedule task: %v", err)
	}
}

func decodeEventData(data []byte, event *JobCreatedEvent) error {
	if len(data) < 160 {
		return fmt.Errorf("invalid event data length")
	}

	// Decode the uint32 jobID
	jobID := new(big.Int).SetBytes(data[:32]).Uint64()
	event.JobID = uint32(jobID)

	// Decode the uint32 timeframe
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

	return nil
}

func (tm *TaskManager) scheduleTask(job JobCreatedEvent) error {
	// Create a cron job to send task to operator
	c := cron.New()

	// Convert timeframe to cron expression
	cronSchedule, err := timeframeToCronExpression(job.Timeframe)
	if err != nil {
		return fmt.Errorf("failed to convert timeframe to cron expression: %v", err)
	}

	// Add the cron job
	err = c.AddFunc(cronSchedule, func() {
		err := sendTaskToOperator(job)
		if err != nil {
			log.Printf("Failed to send task to operator: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %v", err)
	}

	c.Start()
	defer c.Stop()

	return nil
}

func timeframeToCronExpression(timeframe uint32) (string, error) {
	if timeframe <= 0 {
		return "", fmt.Errorf("invalid timeframe: must be greater than zero")
	}

	// Here we assume the timeframe is in seconds for simplicity
	// Adjust the logic according to your specific needs

	// If timeframe is in seconds, we need to convert it to cron format
	// For instance, for a timeframe of 60 seconds, cron expression would be: "* * * * *"
	// For a timeframe of 3600 seconds (1 hour), cron expression would be: "0 * * * *"

	minutes := timeframe / 60
	seconds := timeframe % 60

	if minutes == 0 {
		// Schedule every N seconds
		return fmt.Sprintf("*/%d * * * * *", seconds), nil
	} else if minutes > 0 && minutes < 60 {
		// Schedule every N minutes
		return fmt.Sprintf("*/%d * * * *", minutes), nil
	} else if minutes >= 60 && minutes < 1440 {
		// Schedule every N hours
		hours := minutes / 60
		return fmt.Sprintf("0 */%d * * *", hours), nil
	} else {
		// Schedule every N days
		days := minutes / 1440
		return fmt.Sprintf("0 0 */%d * *", days), nil
	}
}

func sendTaskToOperator(job JobCreatedEvent) error {
	taskJSON, err := json.Marshal(job)
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
