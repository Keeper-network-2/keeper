package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli"
	"github.com/Keeper-network-2/keeper/aggregator"
	"github.com/Layr-Labs/incredible-squaring-avs/core/config"
)

var (
	// Version is the version of the binary.
	Version   string
	GitCommit string
	GitDate   string
)

func main() {
	app := cli.NewApp()
	app.Flags = config.Flags
	app.Version = fmt.Sprintf("%s-%s-%s", Version, GitCommit, GitDate)
	app.Name = "keeper-network-aggregator"
	app.Usage = "Keeper Network Aggregator"
	app.Description = "Service that aggregates responses from Keepers for the Keeper AVS on EigenLayer."
	app.Action = aggregatorMain
	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln("Application failed.", "Message:", err)
	}
}

func aggregatorMain(ctx *cli.Context) error {
	log.Println("Initializing Aggregator")
	config, err := config.NewConfig(ctx)
	if err != nil {
		return err
	}

	configJson, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		config.Logger.Fatalf(err.Error())
	}
	fmt.Println("Config:", string(configJson))

	agg, err := aggregator.NewAggregator(config)
	if err != nil {
		return err
	}

	err = agg.Start(context.Background())
	if err != nil {
		return err
	}

	return nil
}