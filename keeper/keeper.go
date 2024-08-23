package keeper

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	// "os"
	"strings"
    // "fmt"

	logger "github.com/Layr-Labs/eigensdk-go/logging"
	"gopkg.in/yaml.v2"

	"github.com/Keeper-network-2/keeper/aggregator"
	"github.com/Keeper-network-2/keeper/keeper/rpc_client"
	nodeC "github.com/Keeper-network-2/keeper/types"
	// "github.com/Layr-Labs/eigensdk-go/signerv2"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Keeper struct {
    loggerK          logger.Logger
    ethClient        *ethclient.Client
    privateKey       *ecdsa.PrivateKey
    publicKey        *ecdsa.PublicKey
    address          common.Address
    aggregatorClient *rpc_client.AggregatorRpcClient
}

type Config struct {
    EthURL       string `yaml:"ethURL"`
    PrivateKeyHex string `yaml:"private"`
    AggregatorAddr string `yaml:"aggregatorAddr"`
}

type Task struct {
    JobID        uint32 `json:"jobID"`
    TaskID       uint32 `json:"taskID"`
    ChainID      uint   `json:"chainID"`
    ContractAddr string `json:"contractAddr"`
    TargetFunc   string `json:"targetFunc"`
}

func NewKeeper(configPath string) (*Keeper, error) {
    configData, err := ioutil.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %v", err)
    }
    // fmt.Printf("Config data: %s\n", configData)

    var config nodeC.NodeConfig
    err = yaml.Unmarshal(configData, &config)
    if err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    // print the config
    fmt.Printf("Config: %+v\n", config)
    


    
    // return &Keeper{
    //     ethClient:        ethClient,
    //     privateKey:       privateKey,
    //     publicKey:        publicKey,
    //     address:          address,
    //     aggregatorClient: aggregatorClient,
    // }, nil
    return &Keeper{}, nil
}

func (k *Keeper) Start() {
    http.HandleFunc("/executeTask", k.executeTaskHandler)
    log.Println("Starting keeper server on port 8081...")
    log.Fatal(http.ListenAndServe(":8081", nil))
}

func (k *Keeper) executeTaskHandler(w http.ResponseWriter, r *http.Request) {
    var task Task
    if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    log.Printf("Received task: %+v\n", task)
    
    result, err := k.executeTask(task)
    if err != nil {
        log.Printf("Error executing task: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    signedResult, err := k.signResult(result)
    if err != nil {
        log.Printf("Error signing result: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    k.sendToAggregator(task.JobID, signedResult)

    w.WriteHeader(http.StatusOK)
}

func (k *Keeper) executeTask(task Task) ([]byte, error) {
    contractABI, err := k.getContractABI(task.ContractAddr)
    if err != nil {
        return nil, fmt.Errorf("failed to get contract ABI: %v", err)
    }

    result, err := k.executeTargetFunction(task.ContractAddr, task.TargetFunc, contractABI)
    if err != nil {
        return nil, fmt.Errorf("failed to execute target function: %v", err)
    }

    receipt, err := k.sendTransaction(common.HexToAddress(task.ContractAddr), result)
    if err != nil {
        return nil, fmt.Errorf("failed to send transaction: %v", err)
    }

    log.Printf("Transaction receipt: %+v", receipt)

    return result, nil
}

func (k *Keeper) executeTargetFunction(contractAddr, targetFunc string, contractABI *abi.ABI) ([]byte, error) {
    log.Printf("Executing function %s on contract %s", targetFunc, contractAddr)

    address := common.HexToAddress(contractAddr)

    method, exist := contractABI.Methods[targetFunc]
    if !exist {
        return nil, fmt.Errorf("method %s not found in contract ABI", targetFunc)
    }

    data, err := method.Inputs.Pack()
    if err != nil {
        return nil, fmt.Errorf("failed to pack method inputs: %v", err)
    }

    result, err := k.ethClient.CallContract(context.Background(), ethereum.CallMsg{
        From: k.address,
        To:   &address,
        Data: data,
    }, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to call contract: %v", err)
    }

    return result, nil
}

func (k *Keeper) getContractABI(contractAddr string) (*abi.ABI, error) {
    apiURL := fmt.Sprintf("https://api.etherscan.io/api?module=contract&action=getabi&address=%s&apikey=RU9BRI2G4CMTK98GEYSDUI1VXJFHBQY5F6", contractAddr)

    resp, err := http.Get(apiURL)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch ABI: %v", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response body: %v", err)
    }

    var result struct {
        Status  string `json:"status"`
        Message string `json:"message"`
        Result  string `json:"result"`
    }

    err = json.Unmarshal(body, &result)
    if err != nil {
        return nil, fmt.Errorf("failed to parse API response: %v", err)
    }

    if result.Status != "1" {
        return nil, fmt.Errorf("API request failed: %s", result.Message)
    }

    parsedABI, err := abi.JSON(strings.NewReader(result.Result))
    if err != nil {
        return nil, fmt.Errorf("failed to parse ABI JSON: %v", err)
    }

    return &parsedABI, nil
}

func (k *Keeper) sendTransaction(to common.Address, data []byte) (*types.Receipt, error) {
    nonce, err := k.ethClient.PendingNonceAt(context.Background(), k.address)
    if err != nil {
        return nil, fmt.Errorf("failed to get nonce: %v", err)
    }

    gasPrice, err := k.ethClient.SuggestGasPrice(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to suggest gas price: %v", err)
    }

    tx := types.NewTransaction(nonce, to, big.NewInt(0), 300000, gasPrice, data)

    chainID, err := k.ethClient.NetworkID(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to get chain ID: %v", err)
    }

    signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), k.privateKey)
    if err != nil {
        return nil, fmt.Errorf("failed to sign transaction: %v", err)
    }

    err = k.ethClient.SendTransaction(context.Background(), signedTx)
    if err != nil {
        return nil, fmt.Errorf("failed to send transaction: %v", err)
    }

    receipt, err := k.ethClient.TransactionReceipt(context.Background(), signedTx.Hash())
    if err != nil {
        return nil, fmt.Errorf("failed to get transaction receipt: %v", err)
    }

    return receipt, nil
}

func (k *Keeper) signResult(result []byte) ([]byte, error) {
    hash := crypto.Keccak256Hash(result)
    signature, err := crypto.Sign(hash.Bytes(), k.privateKey)
    if err != nil {
        return nil, fmt.Errorf("failed to sign result: %v", err)
    }
    
    return signature, nil
}

func (k *Keeper) sendToAggregator(jobID uint32, signedResult []byte) {
    signedTaskResponse := &aggregator.SignedTaskResponse{
        JobID:      jobID,
        SignedData: signedResult,
    }
    k.aggregatorClient.SendSignedTaskResponseToAggregator(signedTaskResponse)
}
