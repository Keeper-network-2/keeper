package aggregator

import (
	"context"
	// "log"
	"sync"
	"net/http"
	"net/rpc"

	"github.com/Layr-Labs/eigensdk-go/logging"

	// "github.com/Layr-Labs/eigensdk-go/crypto/bls"
	"github.com/Layr-Labs/eigensdk-go/types"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients"
	"github.com/Layr-Labs/eigensdk-go/services/avsregistry"
	blsagg "github.com/Layr-Labs/eigensdk-go/services/bls_aggregation"
	oprsinfoserv "github.com/Layr-Labs/eigensdk-go/services/operatorsinfo"
	sdktypes "github.com/Layr-Labs/eigensdk-go/types"

	"github.com/Layr-Labs/incredible-squaring-avs/core"
	"github.com/Layr-Labs/incredible-squaring-avs/core/chainio"
	"github.com/Layr-Labs/incredible-squaring-avs/core/config"

	cstaskmanager "github.com/Layr-Labs/incredible-squaring-avs/contracts/bindings/IncredibleSquaringTaskManager"
)

type Aggregator struct {
	logger                logging.Logger
	serverIpPortAddr      string
	avsWriter             chainio.AvsWriterer
	blsAggregationService blsagg.BlsAggregationService
	tasks                 map[types.TaskIndex]cstaskmanager.IIncredibleSquaringTaskManagerTask
	tasksMu               sync.RWMutex
	taskResponses         map[types.TaskIndex]map[sdktypes.TaskResponseDigest]cstaskmanager.IIncredibleSquaringTaskManagerTaskResponse
	taskResponsesMu       sync.RWMutex
	shutdownChan          chan struct{}
	wg                    sync.WaitGroup
}

type SignedTaskResponse struct {
	JobID      uint32
	SignedData []byte
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

	operatorPubkeysService := oprsinfoserv.NewOperatorsInfoServiceInMemory(context.Background(), sdkClients.AvsRegistryChainSubscriber, sdkClients.AvsRegistryChainReader, c.Logger)
	avsRegistryService := avsregistry.NewAvsRegistryServiceChainCaller(avsReader, operatorPubkeysService, c.Logger)
	blsAggregationService := blsagg.NewBlsAggregatorService(avsRegistryService, c.Logger)

	return &Aggregator{
		logger:                c.Logger,
		serverIpPortAddr:      c.AggregatorServerIpPortAddr,
		avsWriter:             avsWriter,
		blsAggregationService: blsAggregationService,
		tasks:                 make(map[types.TaskIndex]cstaskmanager.IIncredibleSquaringTaskManagerTask),
		taskResponses:         make(map[types.TaskIndex]map[sdktypes.TaskResponseDigest]cstaskmanager.IIncredibleSquaringTaskManagerTaskResponse),
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

	return nil
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

func (agg *Aggregator) listenForEvents(ctx context.Context) {
	defer agg.wg.Done()
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

func (agg *Aggregator) Shutdown(ctx context.Context) error {
	agg.logger.Info("Shutting down aggregator")
	close(agg.shutdownChan)

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		agg.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}



/*

package main

import (
	"context"
	"net/http"
	"sync"
	"math/big"

	"github.com/Layr-Labs/eigensdk-go/logging"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients"
	sdkclients "github.com/Layr-Labs/eigensdk-go/chainio/clients"
	"github.com/Layr-Labs/eigensdk-go/services/avsregistry"
	blsagg "github.com/Layr-Labs/eigensdk-go/services/bls_aggregation"
	oprsinfoserv "github.com/Layr-Labs/eigensdk-go/services/operatorsinfo"
	sdktypes "github.com/Layr-Labs/eigensdk-go/types"

	"github.com/Layr-Labs/incredible-squaring-avs/aggregator/types"
	"github.com/Layr-Labs/incredible-squaring-avs/core"
	"github.com/Layr-Labs/incredible-squaring-avs/core/chainio"
	"github.com/Layr-Labs/incredible-squaring-avs/core/config"

	cstaskmanager "github.com/Layr-Labs/incredible-squaring-avs/contracts/bindings/IncredibleSquaringTaskManager"
)

type Aggregator struct {
	logger           logging.Logger
	serverIpPortAddr string
	avsWriter        chainio.AvsWriterer
	blsAggregationService blsagg.BlsAggregationService
	tasks                 map[types.TaskIndex]cstaskmanager.IIncredibleSquaringTaskManagerTask
	tasksMu               sync.RWMutex
	taskResponses         map[types.TaskIndex]map[sdktypes.TaskResponseDigest]cstaskmanager.IIncredibleSquaringTaskManagerTaskResponse
	taskResponsesMu       sync.RWMutex
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

	chainioConfig := sdkclients.BuildAllConfig{
		EthHttpUrl:                 c.EthHttpRpcUrl,
		EthWsUrl:                   c.EthWsRpcUrl,
		RegistryCoordinatorAddr:    c.IncredibleSquaringRegistryCoordinatorAddr.String(),
		OperatorStateRetrieverAddr: c.OperatorStateRetrieverAddr.String(),
		AvsName:                    "KeeperNetwork",
		PromMetricsIpPortAddress:   ":9090",
	}
	clients, err := clients.BuildAll(chainioConfig, c.EcdsaPrivateKey, c.Logger)
	if err != nil {
		c.Logger.Errorf("Cannot create sdk clients", "err", err)
		return nil, err
	}

	operatorPubkeysService := oprsinfoserv.NewOperatorsInfoServiceInMemory(context.Background(), clients.AvsRegistryChainSubscriber, clients.AvsRegistryChainReader, c.Logger)
	avsRegistryService := avsregistry.NewAvsRegistryServiceChainCaller(avsReader, operatorPubkeysService, c.Logger)
	blsAggregationService := blsagg.NewBlsAggregatorService(avsRegistryService, c.Logger)

	return &Aggregator{
		logger:                c.Logger,
		serverIpPortAddr:      c.AggregatorServerIpPortAddr,
		avsWriter:             avsWriter,
		blsAggregationService: blsAggregationService,
		tasks:                 make(map[types.TaskIndex]cstaskmanager.IIncredibleSquaringTaskManagerTask),
		taskResponses:         make(map[types.TaskIndex]map[sdktypes.TaskResponseDigest]cstaskmanager.IIncredibleSquaringTaskManagerTaskResponse),
	}, nil
}

func (agg *Aggregator) Start(ctx context.Context) error {
	agg.logger.Info("Starting aggregator.")
	agg.logger.Info("Starting aggregator rpc server.")
	go agg.startServer(ctx)

	// ticker := time.NewTicker(10 * time.Second)
	// defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		// case <-ticker.C:
			//
		}
	}
}

func (agg *Aggregator) startServer(ctx context.Context) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Aggregator is running"))
	})

	server := &http.Server{Addr: agg.serverIpPortAddr}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	agg.logger.Infof("Starting HTTP server at %s", agg.serverIpPortAddr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		agg.logger.Errorf("HTTP server ListenAndServe: %v", err)
	}
}

func (agg *Aggregator) sendAggregatedResponseToContract(blsAggServiceResp blsagg.BlsAggregationServiceResponse) {
    if blsAggServiceResp.Err != nil {
        agg.logger.Error("BlsAggregationServiceResponse contains an error", "err", blsAggServiceResp.Err)
        panic(blsAggServiceResp.Err)
    }

    nonSignerPubkeys := []cstaskmanager.BN254G1Point{}
    for _, nonSignerPubkey := range blsAggServiceResp.NonSignersPubkeysG1 {
        nonSignerPubkeys = append(nonSignerPubkeys, core.ConvertToBN254G1Point(nonSignerPubkey))
    }

    quorumApks := []cstaskmanager.BN254G1Point{}
    for _, quorumApk := range blsAggServiceResp.QuorumApksG1 {
        quorumApks = append(quorumApks, core.ConvertToBN254G1Point(quorumApk))
    }

	nonSignerStakesAndSignature := cstaskmanager.IBLSSignatureCheckerNonSignerStakesAndSignature{
        NonSignerPubkeys:             nonSignerPubkeys,
        QuorumApks:                   quorumApks,
        ApkG2:                        core.ConvertToBN254G2Point(blsAggServiceResp.SignersApkG2),
        Sigma:                        core.ConvertToBN254G1Point(blsAggServiceResp.SignersAggSigG1.G1Point),
        NonSignerQuorumBitmapIndices: blsAggServiceResp.NonSignerQuorumBitmapIndices,
        QuorumApkIndices:             blsAggServiceResp.QuorumApkIndices,
        TotalStakeIndices:            blsAggServiceResp.TotalStakeIndices,
        NonSignerStakeIndices:        blsAggServiceResp.NonSignerStakeIndices,
    }

    agg.logger.Info("Threshold reached. Sending aggregated response onchain.",
        "taskIndex", blsAggServiceResp.TaskIndex,
    )

    agg.tasksMu.RLock()
    task := agg.tasks[blsAggServiceResp.TaskIndex]
    agg.tasksMu.RUnlock()

    agg.taskResponsesMu.RLock()
    taskResponse := agg.taskResponses[blsAggServiceResp.TaskIndex][blsAggServiceResp.TaskResponseDigest]
    agg.taskResponsesMu.RUnlock()

	_, err := agg.avsWriter.SendAggregatedResponse(context.Background(), task, taskResponse, nonSignerStakesAndSignature)
    if err != nil {
        agg.logger.Error("Aggregator failed to respond to task", "err", err)
    }
}


*/