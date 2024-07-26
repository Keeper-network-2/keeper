package main

import (
    "fmt"
    "net/rpc"
    "time"

    "github.com/Keeper-network-2/keeper/aggregator"
)

type AggregatorRpcClienter interface {
    SendSignedTaskResponseToAggregator(signedTaskResponse *aggregator.SignedTaskResponse)
}

type AggregatorRpcClient struct {
    rpcClient            *rpc.Client
    aggregatorIpPortAddr string
}

func NewAggregatorRpcClient(aggregatorIpPortAddr string) (*AggregatorRpcClient, error) {
    return &AggregatorRpcClient{
        rpcClient:            nil,
        aggregatorIpPortAddr: aggregatorIpPortAddr,
    }, nil
}

func (c *AggregatorRpcClient) dialAggregatorRpcClient() error {
    client, err := rpc.DialHTTP("tcp", "localhost"+c.aggregatorIpPortAddr)
    if err != nil {
        return err
    }
    c.rpcClient = client
    return nil
}

func (c *AggregatorRpcClient) SendSignedTaskResponseToAggregator(signedTaskResponse *aggregator.SignedTaskResponse) {
    if c.rpcClient == nil {
        fmt.Println("RPC client is nil. Dialing aggregator RPC client")
        err := c.dialAggregatorRpcClient()
        if err != nil {
            fmt.Printf("Could not dial aggregator RPC client. Not sending signed task response to aggregator. Is aggregator running? Error: %v\n", err)
            return
        }
    }

    var reply bool
    fmt.Printf("Sending signed task response to aggregator: %#v\n", signedTaskResponse)
    
    for i := 0; i < 5; i++ {
        err := c.rpcClient.Call("Aggregator.ProcessSignedTaskResponse", signedTaskResponse, &reply)
        if err != nil {
            fmt.Printf("Received error from aggregator: %v\n", err)
        } else {
            fmt.Printf("Signed task response accepted by aggregator. Reply: %v\n", reply)
            return
        }
        fmt.Println("Retrying in 2 seconds")
        time.Sleep(2 * time.Second)
    }
    
    fmt.Println("Could not send signed task response to aggregator. Tried 5 times.")
}