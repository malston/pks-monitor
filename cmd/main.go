package main

import (
	"context"
	"fmt"
	monitor "github.com/pupimvictor/pks-monitor"
	"io"
	"net/http"
	"os"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"time"
)

func main() {
	cliId := os.Getenv("CLI_ID")
	cliSecret := os.Getenv("CLI_SECRET")
	api := os.Getenv("API")

	if cliId == "" || cliSecret == "" || api == "" {
		fmt.Println("missing api address or uaa client credentials")
		os.Exit(1)
	}

	pksMonitor, _ := monitor.NewPksMonitor(api)

	fmt.Printf("monitoring: %s\n", api)

	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("shutting down...")
				time.Sleep(20 * time.Second)
				os.Exit(1)
			}
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/healthz", healthz)

	go pksMonitor.Run(ctx, cancelFunc, api, cliId)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
		cancelFunc()
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok"}`)
}

