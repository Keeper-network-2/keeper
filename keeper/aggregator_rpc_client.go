package keeper

import (
    "fmt"
    "net/rpc"
    "time"

    "github.com/Keeper-network-2/keeper/aggregator"
    /* "github.com/yourorg/yourproject/metrics"
    "github.com/yourorg/yourproject/logging" */
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
    client, err := rpc.DialHTTP("tcp", c.aggregatorIpPortAddr)
    if err != nil {
        return err
    }
    c.rpcClient = client
    return nil
}

func (c *AggregatorRpcClient) SendSignedTaskResponseToAggregator(signedTaskResponse *aggregator.SignedTaskResponse) {
    if c.rpcClient == nil {
        fmt.Println("rpc client is nil. Dialing aggregator rpc client")
        err := c.dialAggregatorRpcClient()
        if err != nil {
            fmt.Println("Could not dial aggregator rpc client. Not sending signed task response header to aggregator. Is aggregator running?", err)
            return
        }
    }
    var reply bool
    fmt.Printf("Sending signed task response header to aggregator: %#v\n", signedTaskResponse)
    for i := 0; i < 5; i++ {
        err := c.rpcClient.Call("Aggregator.ProcessSignedTaskResponse", signedTaskResponse, &reply)
        if err != nil {
            fmt.Println("Received error from aggregator", err)
        } else {
            fmt.Println("Signed task response header accepted by aggregator.", reply)
            return
        }
        fmt.Println("Retrying in 2 seconds")
        time.Sleep(2 * time.Second)
    }
    fmt.Println("Could not send signed task response to aggregator. Tried 5 times.")
}