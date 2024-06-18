package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
)

type JobCreatedEvent struct {
	JobID          uint32 `json:"jobID"`
	JobType        string `json:"jobType"`
	JobDescription string `json:"jobDescription"`
	JobURL         string `json:"jobURL"`
}

func main() {
	http.HandleFunc("/executeTask", executeTaskHandler)
	log.Println("Starting operator server on port 8081...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func executeTaskHandler(w http.ResponseWriter, r *http.Request) {
	var job JobCreatedEvent
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received task: %+v\n", job)

	// Perform task execution logic here
	executeJob(job)

	w.WriteHeader(http.StatusOK)
}

func executeJob(job JobCreatedEvent) {
	log.Printf("Executing job %d: fetching script from %s", job.JobID, job.JobURL)
	response, err := http.Get(job.JobURL)
	if err != nil {
		log.Printf("Error fetching script for job %d: %v", job.JobID, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Printf("Unexpected status code %d for job %d", response.StatusCode, job.JobID)
		return
	}

	// Use io.ReadAll to read response body
	script, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("Error reading script for job %d: %v", job.JobID, err)
		return
	}

	executeScript(script)
}

func executeScript(script []byte) {
	log.Println("Executing script...")
	cmd := exec.Command("node", "-e", string(script))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error executing script: %v", err)
		return
	}
	log.Printf("Script output:\n%s", output)
}
