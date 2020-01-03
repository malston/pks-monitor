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

type PksMonitor struct {
	pksApi      string
	accessToken string
	client      *http.Client
}

type uaaClient struct {
	clientId     string
	clientSecret string
}

func NewPksMonitor(api string) (*PksMonitor, error) {
	// config for skip SSL verification
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}

	return &PksMonitor{
		pksApi: api,
		client: client,
	}, nil
}

func init() {
	prometheus.MustRegister(pksApiUp)
}

func (pks PksMonitor) Run(ctx context.Context, cancelFunc context.CancelFunc, api, token string) {
	// build api uri to list clusters
	url := fmt.Sprintf("https://%s:9021/v1/clusters", api)
	method := "GET"

monitorLoop:
	for {
		select {
		case <-ctx.Done():
			fmt.Println("stopping running. context is done")
			return

		// executes api request every 10 seconds.
		case <-time.Tick(10 * time.Second):
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
			res, err := pks.client.Do(req)
			if err != nil {
				pksApiUp.Set(0.0)
				fmt.Println(0)
				fmt.Println(err)
				break monitorLoop
			}
			_ = res.Body.Close()

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
