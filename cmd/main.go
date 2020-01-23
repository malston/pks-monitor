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

func main() {
	cliId := os.Getenv("UAA_CLI_ID")
	cliSecret := os.Getenv("UAA_CLI_SECRET")
	api := os.Getenv("PKS_API")

	if cliId == "" || cliSecret == "" || api == "" {
		fmt.Println("main: missing api address or uaa client credentials")
		os.Exit(1)
	}

	//set default api check interval to 30 seconds
	intervalDuration := time.Second * 30

	//check for custom interval
	interval := os.Getenv("API_CHECK_INTERVAL_SECS")
	if interval != "" {
		intervalTime, err := time.ParseDuration(fmt.Sprintf("%s%s", interval, "s"))
		if err != nil {
			intervalDuration = intervalTime
		}
	}

	pksMonitor, err := monitor.NewPksMonitor(api, cliId, cliSecret)
	if err != nil {
		fmt.Printf("main: could not authenticate to api: %+v\n", err)
		log.Fatal(err)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

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
			// executes api request every `intervalDuration` seconds.
			case <-time.Tick(intervalDuration):
				err := pksMonitor.CheckAPI()
				if err != nil {
					fmt.Printf("main: could not check api: %+v\n", err)
				}

			// stop process because server stopped working
			case <-ctx.Done():
				fmt.Println("main: stopping running. server stopped working")
				break monitorLoop

			// stop process because OS signal received
			case sig := <-done:
				fmt.Printf("main: stopping running. received OS sig to stop: %s\n", sig.String())
				break monitorLoop
			}
		}
	}()

	// start http server
	err = srv.ListenAndServe()
	if err != nil {
		fmt.Printf("main: server stopped: %+v", err)
		cancelFunc()
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok"}`)
}

func prestop(w http.ResponseWriter, r *http.Request) {
	fmt.Println("prestop...")
	w.Header().Add("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"shutting down"}`)
}
