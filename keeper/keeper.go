package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "io/ioutil"

    "github.com/joho/godotenv"
    blst "github.com/supranational/blst/bindings/go"
    // "github.com/Keeper-network-2/keeper/keeper"
    "github.com/Keeper-network-2/keeper/aggregator/rpc"
)

type JobCreatedEvent struct {
    JobID          uint32 `json:"jobID"`
    JobType        string `json:"jobType"`
    JobDescription string `json:"jobDescription"`
    JobURL         string `json:"jobURL"`
}



var rpcClient *aggregator_rpc.AggregatorRpcClient

func main() {
    // Load environment variables from .env file
    err := godotenv.Load()
    if (err != nil) {
        log.Fatal("Error loading .env file")
    }

    aggregatorIpPortAddr := os.Getenv("AGGREGATOR_IP_PORT")
    rpcClient, err = aggregator_rpc.NewAggregatorRpcClient(aggregatorIpPortAddr)
    if (err != nil) {
        log.Fatalf("Error creating RPC client: %v", err)
    }

    http.HandleFunc("/executeTask", executeTaskHandler)
    log.Println("Starting operator server on port 8081...")
    log.Fatal(http.ListenAndServe(":8081", nil))
}

func executeTaskHandler(w http.ResponseWriter, r *http.Request) {
    var job JobCreatedEvent
    if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    log.Printf("Received task: %+v\n", job)

    // Perform task execution logic here
    executeJob(job.JobID)

    w.WriteHeader(http.StatusOK)
}

func executeJob(jobID uint32) {
    script, err := ioutil.ReadFile("script.js")
    if (err != nil) {
        log.Printf("Error reading script file: %v", err)
        return
    }

    encodedData := string(script)
    log.Printf("Encoded data from script: %s", encodedData)

    signedData := signJobResult(encodedData)
    sendSignedResultToAggregator(signedData, jobID)
}

func signJobResult(encodedData string) string {
    privateKey := os.Getenv("BLS_PRIVATE_KEY")

    // Sign the encoded data using BLS
    privKey := blst.SecretKey{}
    privKey.DeserializeHexStr(privateKey)

    message := []byte(encodedData)
    signature := privKey.Sign(message, nil)

    signedData := signature.Serialize()
    return fmt.Sprintf("%x", signedData)
}

func sendSignedResultToAggregator(signedData string, jobID uint32) {
    // Construct the signed task response
    signedTaskResponse := &aggregator.SignedTaskResponse{
        JobID:      jobID,
        SignedData: signedData,
    }

    // Send the signed task response to the aggregator
    rpcClient.SendSignedTaskResponseToAggregator(signedTaskResponse)
}