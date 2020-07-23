package monitor

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	pksNet "github.com/pupimvictor/pks-monitor/net"
)

var (
	pksApiUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "wf",
		Subsystem: "opp",
		Name:      "pks_api_up",
		Help:      "Is the Pks Api up?",
	})

	pksListClusters = "/v1/clusters"
)

func init() {
	prometheus.MustRegister(pksApiUp)
}

type PksMonitor struct {
	config *Config
	client *http.Client
}

func NewPksMonitor(api, cliId, cliSecret string) (*PksMonitor, error) {
	// Create a CA certificate pool and add cert.pem to it
	caCert, err := ioutil.ReadFile("/etc/pks-monitor/certs/cert.pem")
	if err != nil {
		return nil, errors.Wrap(err, "monitor: couldn't read certs")
	}

	//check if URL is properly formatted
	u, err := url.Parse(api)
	if err != nil {
		return nil, err
	}
	host := u.Host
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return nil, err
		}
	}

	config := &Config{
		API:                 u.Scheme + "://" + host,
		CACert:              string(caCert),
		SkipSSLVerification: true,
		UaaCliId:            cliId,
		UaaCliSecret:        cliSecret,
	}

	err = AuthenticateApi(config)
	if err != nil {
		return nil, errors.Wrap(err, "monitor: couldn't login to pks")
	}
	//fmt.Printf("fresh token: %+v\n", config.AccessToken)

	client, err := CreateHttpClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "monitor: couldnt't create http client")
	}

	pksMonitor := &PksMonitor{
		client: client,
		config: config,
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
	reqUrl := pks.config.API + ":" + APIPort + pksListClusters

	// create request object
	req, err := http.NewRequest(method, reqUrl, nil)
	if err != nil {
		return false, errors.Wrap(err, "monitor: unable to create new request")
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	// making api request
	res, err := pks.client.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "monitor: unable to make API request")
	}
	defer res.Body.Close()

	// check if api resp error is a expired token and try to reconnect
	if expired, _ := pksNet.TokenExpired(res); expired {
		fmt.Println("reauthenticate...")
		err := AuthenticateApi(pks.config)
		if err != nil {
			return false, errors.Wrap(err, "monitoring: unable to reauthenticate: %+v\n")
		}
		return true, nil
	}

	// check success of api call
	if res.StatusCode != 200 {
		fmt.Printf("monitor: PKS API seems to be down - response status code: %d\n", res.StatusCode)
		return false, nil
	}

	return true, nil
}

func AuthenticateApi(c *Config) error {
	uaaClient, err := CreateUaaClient(c)
	if err != nil {
		return err
	}

	// request for /actuator/info to setup cookies
	request, err := http.NewRequest("HEAD", uaaClient.AuthURL.String()+"/actuator/info", nil)
	if err != nil {
		return errors.Wrap(err, "Unable to create an HTTPS request.")
	}
	response, err := uaaClient.Client.Do(request)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Unable to send a request to API: %s", request.RequestURI))
	}
	if response.StatusCode == http.StatusUnauthorized {
		return errors.New("monitor: AuthenticateApi - Credentials were rejected, please try again.")
	}

	// call uaa api for access token
	token, err := uaaClient.ClientCredentialGrant(c.UaaCliId, c.UaaCliSecret)
	if err != nil {
		return errors.Wrap(err, "monitor: couldn't get token")
	}

	c.AccessToken = token.AccessToken
	return nil
}
