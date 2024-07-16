package main

import (
	"fmt"
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

			fmt.Println("Initializing Task Manager...")
			tm, err := taskmanager.NewTaskManager(clientURL, contractAddr)
			if err != nil {
				fmt.Printf("Error initializing Task Manager: %v\n", err)
				return err
			}

			fmt.Println("Task Manager initialized successfully.")
			fmt.Println("Listening for events...")
			tm.ListenForEvents()
			fmt.Println("Stopped listening for events.")
			return nil
		},
	}

	fmt.Println("Starting task-manager application...")
	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("Application error: %v\n", err)
	}
	fmt.Println("Application stopped.")
}
