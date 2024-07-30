package main

import (
    "log"
    "os"

    "github.com/urfave/cli"
    "github.com/Keeper-network-2/keeper/keeper"
)

func main() {
    app := cli.NewApp()
    app.Name = "keeper"
    app.Usage = "Keeper Node"
    app.Action = runKeeper

    app.Flags = []cli.Flag{
        cli.StringFlag{
            Name:  "eth-url",
            Usage: "Ethereum client URL",
        },
        cli.StringFlag{
            Name:  "aggregator-addr",
            Usage: "Aggregator RPC address",
        },
        cli.StringFlag{
            Name:  "private-key",
            Usage: "Private key in hex format",
        },
    }

    err := app.Run(os.Args)
    if err != nil {
        log.Fatal(err)
    }
}

func runKeeper(ctx *cli.Context) error {
    ethURL := ctx.String("eth-url")
    aggregatorAddr := ctx.String("aggregator-addr")
    privateKeyHex := ctx.String("private-key")

    k, err := keeper.NewKeeper(ethURL, aggregatorAddr, privateKeyHex)
    if err != nil {
        return err
    }

    k.Start()
    return nil
}
