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
	token := os.Getenv("TOKEN")
	api := os.Getenv("API")

	if token == "" || api == "" {
		fmt.Println("no token or api on env")
		os.Exit(1)
	}
	//fmt.Println(token)
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

	go monitor.Run(ctx, cancelFunc, api, token)

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

