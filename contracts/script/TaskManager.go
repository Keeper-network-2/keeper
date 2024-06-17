package main

import (
    "context"
    "log"
    "math/big"
    "time"

    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"
)

type Listener struct {
    logger      *log.Logger
    sendNewTask func(*big.Int) error
}

func (lis *Listener) ListenForJobEvents(ctx context.Context, contractAddress common.Address, rpcClient *rpc.Client) error {
   /*  ticker := time.NewTicker(10 * time.Second)
    lis.logger.Println("Listener set to send new task every 10 seconds...")
    defer ticker.Stop()
    taskNum := int64(0)

    // Send the first task immediately
    _ = lis.sendNewTask(big.NewInt(taskNum))
    taskNum++
 */
    // Set up the Ethereum client
    client := ethclient.NewClient(rpcClient)

    // Subscribe to the IncredibleSquaringTaskManager contract events
    query := ethereum.FilterQuery{
        Addresses: []common.Address{contractAddress},
    }
    logs := make(chan types.Log)
    sub, err := client.SubscribeFilterLogs(ctx, query, logs)
    if err != nil {
        return err
    }
    defer sub.Unsubscribe()

    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            err := lis.sendNewTask(big.NewInt(taskNum))
            taskNum++
            if err != nil {
                lis.logger.Println("Error sending new task:", err)
            }
        case vLog := <-logs:
            lis.logger.Println("Received IncredibleSquaringTaskManager event", "vLog", vLog)
            // Process the event and send a new task
            err := lis.sendNewTask(big.NewInt(taskNum))
            taskNum++
            if err != nil {
                lis.logger.Println("Error sending new task:", err)
            }
        case err := <-sub.Err():
            return err
        }
    }
}