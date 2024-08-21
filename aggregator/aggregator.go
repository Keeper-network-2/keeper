package aggregator

import (
	"context"
	"math/big"
	// "net/http"
	// "net/rpc"
	"sync"

	"github.com/Layr-Labs/eigensdk-go/logging"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients"
	"github.com/Layr-Labs/eigensdk-go/services/avsregistry"
	blsagg "github.com/Layr-Labs/eigensdk-go/services/bls_aggregation"
	oprsinfoserv "github.com/Layr-Labs/eigensdk-go/services/operatorsinfo"
	sdktypes "github.com/Layr-Labs/eigensdk-go/types"

	"github.com/Keeper-network-2/keeper/core"
	"github.com/Keeper-network-2/keeper/core/chainio"
	"github.com/Keeper-network-2/keeper/core/config"
	"github.com/Keeper-network-2/keeper/core/types"

	cstaskmanager "github.com/Layr-Labs/incredible-squaring-avs/contracts/bindings/IncredibleSquaringTaskManager"
)

type Aggregator struct {
	logger                logging.Logger
	serverIpPortAddr      string
	avsWriter             chainio.AvsWriterer
	blsAggregationService blsagg.BlsAggregationService
	tasks                 map[sdktypes.TaskIndex]cstaskmanager.IIncredibleSquaringTaskManagerTask
	tasksMu               sync.RWMutex
	taskResponses         map[sdktypes.TaskIndex]map[sdktypes.TaskResponseDigest]cstaskmanager.IIncredibleSquaringTaskManagerTaskResponse
	taskResponsesMu       sync.RWMutex
	shutdownChan          chan struct{}
	wg                    sync.WaitGroup
}



func NewAggregator(c *config.Config) (*Aggregator, error) {
	avsReader, err := chainio.BuildAvsReaderFromConfig(c)
	if err != nil {
		c.Logger.Error("Cannot create avsReader", "err", err)
		return nil, err
	}

	avsWriter, err := chainio.BuildAvsWriterFromConfig(c)
	if err != nil {
		c.Logger.Errorf("Cannot create avsWriter", "err", err)
		return nil, err
	}

	chainioConfig := clients.BuildAllConfig{
		EthHttpUrl:                 c.EthHttpRpcUrl,
		EthWsUrl:                   c.EthWsRpcUrl,
		RegistryCoordinatorAddr:    c.IncredibleSquaringRegistryCoordinatorAddr.String(),
		OperatorStateRetrieverAddr: c.OperatorStateRetrieverAddr.String(),
		AvsName:                    "KeeperNetwork",
		PromMetricsIpPortAddress:   ":9090",
	}

	sdkClients, err := clients.BuildAll(chainioConfig, c.EcdsaPrivateKey, c.Logger)
	if err != nil {
		c.Logger.Errorf("Cannot create sdk clients", "err", err)
		return nil, err
	}

	logFilterQueryBlockRange := big.NewInt(100)

	hashFunction := func(taskResponse sdktypes.TaskResponse) (sdktypes.TaskResponseDigest, error) {
		customResponse, _ := taskResponse.(types.TaskResponse)
		hash := crypto.Keccak256Hash(customResponse.Result)
		return sdktypes.TaskResponseDigest(hash), nil
	}
	operatorPubkeysService := oprsinfoserv.NewOperatorsInfoServiceInMemory(context.Background(), sdkClients.AvsRegistryChainSubscriber, sdkClients.AvsRegistryChainReader, logFilterQueryBlockRange, c.Logger)
	avsRegistryService := avsregistry.NewAvsRegistryServiceChainCaller(avsReader, operatorPubkeysService, c.Logger)
	blsAggregationService := blsagg.NewBlsAggregatorService(avsRegistryService, hashFunction, c.Logger)

	return &Aggregator{
		logger:                c.Logger,
		serverIpPortAddr:      c.AggregatorServerIpPortAddr,
		avsWriter:             avsWriter,
		blsAggregationService: blsAggregationService,
		tasks:                 make(map[sdktypes.TaskIndex]cstaskmanager.IIncredibleSquaringTaskManagerTask),
		taskResponses:         make(map[sdktypes.TaskIndex]map[sdktypes.TaskResponseDigest]cstaskmanager.IIncredibleSquaringTaskManagerTaskResponse),
	}, nil
}

func (agg *Aggregator) Start(ctx context.Context) error {
	agg.logger.Info("Starting aggregator")

	agg.wg.Add(2)
	go func() {
		defer agg.wg.Done()
		err := agg.startServer(ctx)
		if err != nil {
			agg.logger.Fatal("Failed to start server: ", err)
		}
	}()

	go func() {
		defer agg.wg.Done()
		agg.listenForEvents(ctx)
	}()

	<-ctx.Done()
	return nil
}



func (agg *Aggregator) listenForEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case blsAggServiceResp := <-agg.blsAggregationService.GetResponseChannel():
			agg.handleAggregatedSignature(blsAggServiceResp)
		}
	}
}

func (agg *Aggregator) handleAggregatedSignature(resp blsagg.BlsAggregationServiceResponse) {
	// Check for errors in the response
	if resp.Err != nil {
		agg.logger.Error("BlsAggregationServiceResponse contains an error", "err", resp.Err)
		// Panicking to help with debugging (fail fast), but shouldn't panic if we run this in production
		panic(resp.Err)
	}

	// Convert non-signer public keys
	nonSignerPubkeys := []cstaskmanager.BN254G1Point{}
	for _, nonSignerPubkey := range resp.NonSignersPubkeysG1 {
		nonSignerPubkeys = append(nonSignerPubkeys, core.ConvertToBN254G1Point(nonSignerPubkey))
	}

	// Convert quorum aggregate public keys
	quorumApks := []cstaskmanager.BN254G1Point{}
	for _, quorumApk := range resp.QuorumApksG1 {
		quorumApks = append(quorumApks, core.ConvertToBN254G1Point(quorumApk))
	}

	// Create non-signer stakes and signature
	nonSignerStakesAndSignature := cstaskmanager.IBLSSignatureCheckerNonSignerStakesAndSignature{
		NonSignerPubkeys:             nonSignerPubkeys,
		QuorumApks:                   quorumApks,
		ApkG2:                        core.ConvertToBN254G2Point(resp.SignersApkG2),
		Sigma:                        core.ConvertToBN254G1Point(resp.SignersAggSigG1.G1Point),
		NonSignerQuorumBitmapIndices: resp.NonSignerQuorumBitmapIndices,
		QuorumApkIndices:             resp.QuorumApkIndices,
		TotalStakeIndices:            resp.TotalStakeIndices,
		NonSignerStakeIndices:        resp.NonSignerStakeIndices,
	}

	agg.logger.Info("Threshold reached. Sending aggregated response on-chain.",
		"taskIndex", resp.TaskIndex,
	)

	// Get the task and task response from the aggregator's maps
	agg.tasksMu.RLock()
	task := agg.tasks[resp.TaskIndex]
	agg.tasksMu.RUnlock()
	agg.taskResponsesMu.RLock()
	taskResponse := agg.taskResponses[resp.TaskIndex][resp.TaskResponseDigest]
	agg.taskResponsesMu.RUnlock()

	// Send the aggregated response on-chain
	_, err := agg.avsWriter.SendAggregatedResponse(context.Background(), task, taskResponse, nonSignerStakesAndSignature)
	if err != nil {
		agg.logger.Error("Aggregator failed to respond to task", "err", err)
	}
}

func (a *Aggregator) DummyMethod(argType *struct{}, replyType *struct{}) error {
    // No operation performed.
    return nil
}