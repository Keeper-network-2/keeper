package aggregator

import (
	"context"
	"errors"
	"net/http"
	"net/rpc"

    // "github.com/Keeper-network-2/keeper/core"

	// "github.com/Layr-Labs/eigensdk-go/crypto/bls"
	// "github.com/Layr-Labs/eigensdk-go/types"
)

var (
	TaskNotFoundError400                     = errors.New("400. Task not found")
	OperatorNotPartOfTaskQuorum400           = errors.New("400. Operator not part of quorum")
	TaskResponseDigestNotFoundError500       = errors.New("500. Failed to get task response digest")
	UnknownErrorWhileVerifyingSignature400   = errors.New("400. Failed to verify signature")
	SignatureVerificationFailed400           = errors.New("400. Signature verification failed")
	CallToGetCheckSignaturesIndicesFailed500 = errors.New("500. Failed to get check signatures indices")
)

type SignedTaskResponse struct {
	JobID      uint32
	SignedData []byte
}

type AggregatorRpcClient struct {
    rpcClient            *rpc.Client
    aggregatorIpPortAddr string
}

func (agg *Aggregator) startServer(ctx context.Context) error {
	err := rpc.Register(agg)
	if err != nil {
		agg.logger.Fatal("Format of service TaskManager isn't correct. ", "err", err)
	}
	rpc.HandleHTTP()
	err = http.ListenAndServe(agg.serverIpPortAddr, nil)
	if err != nil {
		agg.logger.Fatal("ListenAndServe", "err", err)
	}

	return nil
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

