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
            Name:  "config",
            Usage: "Path to Config File",
        },
    }

    err := app.Run(os.Args)
    if err != nil {
        log.Fatal(err)
    }
}

func runKeeper(ctx *cli.Context) error {
    configPath := ctx.String("config")

    k, err := keeper.NewKeeper(configPath)
    if err != nil {
        return err
    }

    k.Start()
    return nil
}
