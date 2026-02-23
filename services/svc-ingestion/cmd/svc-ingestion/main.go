package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/runtime"
)

const (
	healthcheckTimeout = 3 * time.Second
	defaultPort        = "8080"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	runtime.New().Run()
}

// runHealthcheck probes the liveness endpoint and returns 0 (healthy) or 1 (unhealthy).
func runHealthcheck() int {
	port := os.Getenv("EDIFACT_SERVER_PORT")
	if port == "" {
		port = defaultPort
	}

	client := &http.Client{Timeout: healthcheckTimeout}

	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/v1/liveness", port))
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return 0
	}

	return 1
}
