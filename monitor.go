package monitor

import (
	"crypto/tls"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
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

type PksMonitor struct {
	apiAddress  string
	accessToken string
	client      *http.Client
}

type uaaClient struct {
	clientId     string
	clientSecret string
}

func NewPksMonitor(api string) (*PksMonitor, error) {
	// config for skip SSL verification
	// TODO: Support TLS
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}

	return &PksMonitor{
		apiAddress: api,
		client:     client,
	}, nil
}

func (pks PksMonitor) CallApi() error {
	// build api uri to list clusters
	url := fmt.Sprintf("https://%s:9021/v1/clusters", pks.apiAddress)
	method := "GET"

	// create request object
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		pksApiUp.Set(0.0)
		return err
	}

	// build headers
	auth := fmt.Sprintf("Bearer %s", pks.accessToken)
	req.Header.Add("Authorization", auth)

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	// making api request
	res, err := pks.client.Do(req)
	if err != nil {
		pksApiUp.Set(0.0)
		return err
	}
	defer res.Body.Close()

	fmt.Printf("response_code: %d\n", res.StatusCode)

	// check success of api call
	if res.StatusCode != 200 {
		pksApiUp.Set(0.0)
		return nil
	}
	// set pks api metric to 1
	pksApiUp.Set(1.0)

	return nil
}
