package monitor

import (
	"crypto/tls"
	"crypto/x509"
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

	pksApiClusters = ":9021/v1/clusters"
	pksApiAuth     = ":8443/oauth/token"
)

func init() {
	prometheus.MustRegister(pksApiUp)
}

type PksMonitor struct {
	pksApiAddr  string
	authApiAddr string
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
	// Create a CA certificate pool and add cert.pem to it
	caCert, err := ioutil.ReadFile("/etc/pks-monitor/certs/cert.pem")
	if err != nil {
		return nil, errors.WithMessage(err, "monitor: couldn't read certs")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	pksMonitor := &PksMonitor{
		pksApiAddr:   fmt.Sprintf("%s%s", api, pksApiClusters),
		authApiAddr:  fmt.Sprintf("%s%s", api, pksApiAuth),
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

	fmt.Printf("monitoring: %s - %s\n", api, time.Now().Format("2006-01-02 15:04:05"))

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

func (pks *PksMonitor) callApi() (bool, error) {
	method := "GET"

	// create request object
	req, err := http.NewRequest(method, pks.pksApiAddr, nil)
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
		fmt.Printf("monitor: error calling pks api. status code: %d\n", res.StatusCode)
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
	oauthParams := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=client_credentials&token_format=opaque", pks.uaaCliId, pks.uaaCliSecret)
	payload := strings.NewReader(oauthParams)
	method := "POST"

	// create request object
	req, err := http.NewRequest(method, pks.authApiAddr, payload)
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

	fmt.Printf("authenticated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	t = &token
	return t, nil
}
