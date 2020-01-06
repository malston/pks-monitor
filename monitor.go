package monitor

import (
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
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

	pksApiClusters = ":9021/v1/clusters"
)

func init() {
	prometheus.MustRegister(pksApiUp)
}

type PksMonitor struct {
	apiAddress  string
	accessToken string
	client      *http.Client

	uaaCliId     string
	uaaCliSecret string
}

func NewPksMonitor(api, cliId, cliSecret string) (*PksMonitor, error) {
	// config for skip SSL verification
	// TODO: Support TLS
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}

	return &PksMonitor{
		apiAddress: api,
		client:     client,
		uaaCliId: cliId,
		uaaCliSecret: cliSecret,
	}, nil
}

func (pks PksMonitor) CheckAPI() error {
	ok, err := pks.callApi()
	if ok {
		pksApiUp.Set(1.0)
	} else {
		pksApiUp.Set(0.0)
	}
	return errors.Wrap(err, "monitor: unable to call API")
}

func (pks PksMonitor) callApi() (bool, error) {
	// build api uri to list clusters
	url := fmt.Sprintf("%s%s", pks.apiAddress, pksApiClusters)
	method := "GET"

	// create request object
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return false, errors.Wrap(err, "monitor: unable to create new request")
	}

	// build headers
	auth := fmt.Sprintf("Bearer %s", pks.accessToken)
	req.Header.Add("Authorization", auth)

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	// making api request
	res, err := pks.client.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "monitor: unable to make API request")
	}
	defer res.Body.Close()

	fmt.Printf("response_code: %d\n", res.StatusCode)

	// check success of api call
	if res.StatusCode != 200 {
		return false, nil
	}

	return true, nil
}


