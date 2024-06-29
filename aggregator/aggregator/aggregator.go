package aggregator

import (
    "context"
    //"math/big"
    "net/http"
    "sync"
    "time"

    "github.com/Layr-Labs/eigensdk-go/logging"
    "github.com/Layr-Labs/eigensdk-go/chainio/clients"
    sdkclients "github.com/Layr-Labs/eigensdk-go/chainio/clients"
    "github.com/Layr-Labs/eigensdk-go/services/avsregistry"
    blsagg "github.com/Layr-Labs/eigensdk-go/services/bls_aggregation"
    oprsinfoserv "github.com/Layr-Labs/eigensdk-go/services/operatorsinfo"
    sdktypes "github.com/Layr-Labs/eigensdk-go/types"
    "aggregator/types"
    "aggregator/core"
    "aggregator/core/chainio"
    "aggregator/core/config"
    cstaskmanager "aggregator/contracts/bindings/IncredibleSquaringTaskManager"
)

const (
    taskChallengeWindowBlock = 100
    blockTimeSeconds         = 12 * time.Second
    avsName                  = "incredible-squaring"
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
        AvsName:                    avsName,
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

func (agg *Aggregator) Start(ctx context.Context) error {
    agg.logger.Infof("Starting aggregator.")
    agg.logger.Infof("Starting aggregator rpc server.")
    go agg.startServer(ctx)

   /*  ticker := time.NewTicker(10 * time.Second)
    agg.logger.Infof("Aggregator set to send new task every 10 seconds...")
    defer ticker.Stop()
    taskNum := int64(0)
    _ = agg.sendNewTask(big.NewInt(taskNum))
    taskNum++

    for {
        select {
        case <-ctx.Done():
            return nil
        case blsAggServiceResp := <-agg.blsAggregationService.GetResponseChannel():
            agg.logger.Info("Received response from blsAggregationService", "blsAggServiceResp", blsAggServiceResp)
            agg.sendAggregatedResponseToContract(blsAggServiceResp)
        case <-ticker.C:
            err := agg.sendNewTask(big.NewInt(taskNum))
            taskNum++
            if err != nil {
                continue
            }
        }
    } */
	<-ctx.Done()

    // Return nil since there's no specific error condition
    return nil
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

/* func (agg *Aggregator) sendNewTask(numToSquare *big.Int) error {
    agg.logger.Info("Aggregator sending new task", "numberToSquare", numToSquare)
    newTask, taskIndex, err := agg.avsWriter.SendNewTaskNumberToSquare(context.Background(), numToSquare, types.QUORUM_THRESHOLD_NUMERATOR, types.QUORUM_NUMBERS)
    if err != nil {
        agg.logger.Error("Aggregator failed to send number to square", "err", err)
        return err
    }

    agg.tasksMu.Lock()
    agg.tasks[taskIndex] = newTask
    agg.tasksMu.Unlock()

    quorumThresholdPercentages := make(sdktypes.QuorumThresholdPercentages, len(newTask.QuorumNumbers))
    for i := range newTask.QuorumNumbers {
        quorumThresholdPercentages[i] = sdktypes.QuorumThresholdPercentage(newTask.QuorumThresholdPercentage)
    }
    taskTimeToExpiry := taskChallengeWindowBlock * blockTimeSeconds
    var quorumNums sdktypes.QuorumNums
    for _, quorumNum := range newTask.QuorumNumbers {
        quorumNums = append(quorumNums, sdktypes.QuorumNum(quorumNum))
    }
    agg.blsAggregationService.InitializeNewTask(taskIndex, newTask.TaskCreatedBlock, quorumNums, quorumThresholdPercentages, taskTimeToExpiry)
    return nil
} */
