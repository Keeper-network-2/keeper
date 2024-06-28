package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "math/big"
    "net/http"
    "os/exec"
    "os"
    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/joho/godotenv"
)

type JobCreatedEvent struct {
    JobID          uint32 `json:"jobID"`
    JobType        string `json:"jobType"`
    JobDescription string `json:"jobDescription"`
    JobURL         string `json:"jobURL"`
}

func main() {
    // Load environment variables from .env file
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
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
    executeJob(job)

    w.WriteHeader(http.StatusOK)
}

func executeJob(job JobCreatedEvent) {
    log.Printf("Executing job %d: fetching script from %s", job.JobID, job.JobURL)
    response, err := http.Get(job.JobURL)
    if err != nil {
        log.Printf("Error fetching script for job %d: %v", job.JobID, err)
        return
    }
    defer response.Body.Close()

    if response.StatusCode != http.StatusOK {
        log.Printf("Unexpected status code %d for job %d", response.StatusCode, job.JobID)
        return
    }

    // Use io.ReadAll to read response body
    script, err := io.ReadAll(response.Body)
    if err != nil {
        log.Printf("Error reading script for job %d: %v", job.JobID, err)
        return
    }

    executeScript(script, job.JobID)
}

func executeScript(script []byte, jobID uint32) {
    log.Println("Executing script...")
    cmd := exec.Command("node", "-e", string(script))
    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Printf("Error executing script: %v", err)
        return
    }
    log.Printf("Script output:\n%s", output)

    // Assuming the script outputs the encoded data, parse it
    var encodedData string
    err = json.Unmarshal(output, &encodedData)
    if err != nil {
        log.Printf("Error parsing script output: %v", err)
        return
    }

    sendTransaction(encodedData, jobID)
}

func sendTransaction(encodedData string, jobID uint32) {
    // Load private key and public address from environment variables
    privateKey := os.Getenv("PRIVATE_KEY")
    publicAddress := os.Getenv("PUBLIC_ADDRESS")

    client, err := ethclient.Dial("https://eth-sepolia.g.alchemy.com/v2/4-Mm4R00QrwNs-Z_vjOPXjSBfO4m4mmV")
    if err != nil {
        log.Fatalf("Failed to connect to the Ethereum client: %v", err)
    }

    privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
    if err != nil {
        log.Fatalf("Failed to load private key: %v", err)
    }

    fromAddress := common.HexToAddress(publicAddress)
    nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
    if err != nil {
        log.Fatalf("Failed to get nonce: %v", err)
    }

    value := big.NewInt(0) // in wei
    gasLimit := uint64(21000) // in units
    gasPrice, err := client.SuggestGasPrice(context.Background())
    if err != nil {
        log.Fatalf("Failed to suggest gas price: %v", err)
    }

    toAddress := common.HexToAddress("0xae1d08dc221640292b93cb46c2502cee6fe68dcc")
    tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, common.FromHex(encodedData))

    chainID, err := client.NetworkID(context.Background())
    if err != nil {
        log.Fatalf("Failed to get chain ID: %v", err)
    }

    signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKeyECDSA)
    if err != nil {
        log.Fatalf("Failed to sign transaction: %v", err)
    }

    err = client.SendTransaction(context.Background(), signedTx)
    if err != nil {
        log.Fatalf("Failed to send transaction: %v", err)
    }

    fmt.Printf("Transaction sent: %s\n", signedTx.Hash().Hex())
}
