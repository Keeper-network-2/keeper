package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"taskmanager/taskmanager"
)

func main() {
	app := &cli.App{
		Name:  "task-manager",
		Usage: "Listen for USDC transfer events and allocate tasks to operators",
		Action: func(c *cli.Context) error {
			clientURL := "ws://localhost:8545"
			contractAddr := "0x9E545E3C0baAB3E08CdfD552C960A1050f373042"

			tm, err := taskmanager.NewTaskManager(clientURL, contractAddr)
			if err != nil {
				return err
			}

			tm.ListenForEvents()
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}






