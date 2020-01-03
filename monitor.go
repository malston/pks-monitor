package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"
)

var (
	pksApiUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "wf",
		Subsystem: "opp",
		Name:      "pks_api_up",
		Help:      "Is the Pks Api up?",
	})
)

func init() {
	prometheus.MustRegister(pksApiUp)
}

func Run(ctx context.Context, cancelFunc context.CancelFunc, api, token string) {
	// build api uri to list clusters
	url := fmt.Sprintf("https://%s:9021/v1/clusters", api)
	method := "GET"

	// config for skip SSL verification
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}

monitorLoop:
	for {
		select {
		case <- ctx.Done():
			fmt.Println("stopping running. context is done")
			return

		// executes api request every 10 seconds.
		case <- time.Tick(10 * time.Second):
			// create request object
			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				pksApiUp.Set(0.0)
				fmt.Println(err)
				fmt.Println(0)
				break monitorLoop
			}

			// building headers
			auth := fmt.Sprintf("Bearer %s", token)
			req.Header.Add("Authorization", auth)

			req.Header.Add("Accept", "application/json")
			req.Header.Add("Content-Type", "application/json")

			// making api request
			res, err := client.Do(req)
			if err != nil {
				pksApiUp.Set(0.0)
				fmt.Println(0)
				fmt.Println(err)
				break monitorLoop
			}
			_ = res.Body.Close()

			/*
			bodyBytes, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Fatal(err)
			}
			bodyString := string(bodyBytes)
			fmt.Println(bodyString)
			*/
			fmt.Printf("response_code: %d\n", res.StatusCode)

			// check success of api call
			if res.StatusCode != 200 {
				pksApiUp.Set(0.0)
				continue monitorLoop
			}
			// set pks api metric to 1
			pksApiUp.Set(1.0)
		}
	}
	cancelFunc()
}

