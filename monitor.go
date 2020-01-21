package monitor

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	pksApiUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "wf",
		Subsystem: "opp",
		Name:      "pks_api_up",
		Help:      "Is the Pks Api up?",
	})
	pksApiLatency = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "wf",
		Subsystem: "opp",
		Name:      "pks_api_latency",
		Help:      "Duration time for a API call",
	})

	pksApiClusters = ":9021/v1/clusters"
	pksApiAuth     = ":8443/oauth/token"
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

type token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	Jti         string `json:"jti"`
}

func NewPksMonitor(api, cliId, cliSecret string) (*PksMonitor, error) {
	// config for skip SSL verification
	// TODO: Support TLS
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}

	pksMonitor := &PksMonitor{
		apiAddress:   api,
		client:       client,
		uaaCliId:     cliId,
		uaaCliSecret: cliSecret,
	}

	token, err := pksMonitor.authenticateApi()
	if err != nil {
		fmt.Printf("could not authenticate: %+v\n", err)
	}

	if token != nil {
		pksMonitor.accessToken = token.AccessToken
	}

	fmt.Printf("monitoring: %s\n", api)

	return pksMonitor, nil
}

// CheckAPI will call the Api and set the prometheus metrics accordingly to it's response
func (pks PksMonitor) CheckAPI() error {
	ok, err := pks.callApi()
	if ok {
		pksApiUp.Set(1.0)
	} else {
		pksApiUp.Set(0.0)
	}
	fmt.Printf("pks api is up: %t\n", ok)

	return errors.Wrap(err, "monitor: unable to call API")
}

func (pks PksMonitor) callApi() (bool, error) {
	start := time.Now()
	defer func() {
		pksApiLatency.Set(float64(time.Since(start).Milliseconds()))
	}()

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

	// check if api resp error is a expired token and try to reconnect
	if res.StatusCode == 401 || res.StatusCode == 403 {
		fmt.Println("reauthenticate...")
		token, err := pks.authenticateApi()
		if err != nil {
			return false, err
		}
		pks.accessToken = token.AccessToken
	}

	// check success of api call
	if res.StatusCode != 200 {
		fmt.Printf("moniotr: error calling pks api. status code: %d\n", res.StatusCode)
		return false, nil
	}

	return true, nil
}

func (pks PksMonitor) authenticateApi() (t *token, err error) {
	defer func() {
		if err != nil {
			pksApiUp.Set(0.0)
		}
	}()

	// build api uri to authenticate
	url := fmt.Sprintf("%s%s", pks.apiAddress, pksApiAuth)
	method := "POST"

	oauthParams := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=client_credentials&token_format=opaque", pks.uaaCliId, pks.uaaCliSecret)
	payload := strings.NewReader(oauthParams)

	// create request object
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, errors.Wrap(err, "monitor: unable to create new request")
	}

	req.Header.Add("Accept", " application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// making api request
	res, err := pks.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "monitor: unable to make API request")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	// check success of api call
	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("monitor: unable to get API access token: %v - %s", res.Status, body))
	}

	//unmarshal json resp body to token object
	var token token
	err = json.Unmarshal(body, &token)
	if err != nil {
		return nil, errors.WithMessage(err, "monitor: couldn't unmarshal token response")
	}
	t = &token
	return t, nil
}
