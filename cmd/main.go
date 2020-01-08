package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pupimvictor/pks-monitor"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var globalCancel context.CancelFunc

func main() {
	cliId := os.Getenv("UAA_CLI_ID")
	cliSecret := os.Getenv("UAA_CLI_SECRET")
	api := os.Getenv("PKS_API")

	if cliId == "" || cliSecret == "" || api == "" {
		fmt.Println("missing api address or uaa client credentials")
		os.Exit(1)
	}

	pksMonitor, err := monitor.NewPksMonitor(api, cliId, cliSecret)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	globalCancel = cancelFunc

	// setup http server
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	router.HandleFunc("/healthz", healthz)
	router.HandleFunc("/prestop", prestop)
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// create OS signal chan for interruptions
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
	monitorLoop:
		for {
			select {
			// executes api request every 30 seconds.
			case <-time.Tick(30 * time.Second):
				err := pksMonitor.CheckAPI()
				if err != nil {
					fmt.Printf("%v\n", err)
					break monitorLoop
				}

			// stop process because server stopped working
			case <-ctx.Done():
				fmt.Println("stopping running. server stopped working")
				break monitorLoop

			// stop process because OS signal received
			case sig := <-done:
				fmt.Printf("stopping running. received OS sig to stop: %s\n", sig.String())
				break monitorLoop
			}
		}

		err := srv.Shutdown(ctx)
		if err != nil {
			fmt.Printf("Server Shutdown Failed: %v\n", err)
			os.Exit(1)
		}
	}()

	// start http server
	err = srv.ListenAndServe()
	if err != nil {
		fmt.Println(err)
		cancelFunc()
	}

	fmt.Println("gracefully shutting down...")
	time.Sleep(30 * time.Second)
	fmt.Println("bye...")
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok"}`)
}

func prestop(w http.ResponseWriter, r *http.Request) {
	fmt.Println("prestop")
	globalCancel()
	w.Header().Add("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"shutting down"}`)
}
