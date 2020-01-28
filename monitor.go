package monitor

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	pksNet "github.com/pupimvictor/pks-monitor/net"
	"github.com/pupimvictor/pks-monitor/uaa"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
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

	pksListClusters = "/v1/clusters"
)

func init() {
	prometheus.MustRegister(pksApiUp)
}

type PksMonitor struct {
	config      *Config
	client      *http.Client

}

func NewPksMonitor(api, cliId, cliSecret string) (*PksMonitor, error) {
	// Create a CA certificate pool and add cert.pem to it
	caCert, err := ioutil.ReadFile("/etc/pks-monitor/certs/cert.pem")
	if err != nil {
		return nil, errors.WithMessage(err, "monitor: couldn't read certs")
	}

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

	token, extraHeaders, err := PksLogin(config)
	if err != nil {
		return nil, errors.WithMessage(err, "monitor: couldn't login to pks")
	}

	//fmt.Printf("pkslogin token: %+v\n", token)

	config.AccessToken = token.AccessToken
	config.RefreshToken = ""
	config.Headers = extraHeaders

	client, err := CreateHttpClient(config)
	if err != nil {
		return nil , errors.WithMessage( err, "monitor: couldnt't create http client")
	}

	pksMonitor := &PksMonitor{
		client:       client,
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

	// build headers
	//auth := fmt.Sprintf("Bearer %s", pks.config.AccessToken)
	//req.Header.Add("Authorization", auth)

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	if pks.config.Headers != nil {
		for name, value := range *pks.config.Headers {
			req.Header.Add(name, strings.Join(value, " "))
		}
	}
	// making api request
	res, err := pks.client.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "monitor: unable to make API request")
	}
	defer res.Body.Close()

	// check if api resp error is a expired token and try to reconnect
	if res.StatusCode == 401 || res.StatusCode == 403 {
		fmt.Println("reauthenticate...")
		token, extraHeaders, err := PksLogin(pks.config)
		if err != nil {
			fmt.Printf("monitoring: unable to reauthenticate: %+v\n", err)
			return false, err
		}

		fmt.Printf("fresh token: %+v\n", token)
		pks.config.AccessToken = token.AccessToken
		pks.config.RefreshToken = token.RefreshToken
		pks.config.Headers = extraHeaders

		return true, nil
	}

	// check success of api call
	if res.StatusCode != 200 {
		fmt.Printf("monitor: error calling pks api. status code: %d\n", res.StatusCode)
		return false, nil
	}

	return true, nil
}

func PksLogin(c *Config) (uaa.Token, *http.Header, error) {
	uaaHTTPClient, err := pksNet.HTTPClient(c.SkipSSLVerification, []byte(c.CACert))
	if err != nil {
		return uaa.Token{}, nil, err
	}
	u, err := url.Parse(c.API)
	if err != nil {
		return uaa.Token{}, nil, err
	}
	host := u.Host
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return uaa.Token{}, nil, err
		}
	}
	uaaClient := &uaa.Client{
		AuthURL:    u.Scheme + "://" + host + ":" + UAAPort,
		Client: uaaHTTPClient,
	}

	request, err := http.NewRequest("HEAD", u.String()+ ":" + UAAPort + "/actuator/info", nil)
	if err != nil {
		return uaa.Token{}, nil, errors.New("Unable to create an HTTPS request.")
	}

	response, err := uaaClient.Client.Do(request)
	if err != nil {
		return uaa.Token{}, nil, fmt.Errorf("Unable to send a request to API: %s", err)
	}
	if response.StatusCode == http.StatusUnauthorized {
		return uaa.Token{}, nil, errors.New("Credentials were rejected, please try again.")
	}
	uaaClient.Client.Jar.SetCookies(u, response.Cookies())

	token, err := uaaClient.ClientCredentialGrant(c.UaaCliId, c.UaaCliSecret)
	if err != nil {
		return uaa.Token{}, nil, errors.WithMessage(err,"monitor: couldn't get token")
	}

	return token, nil, nil
}

//func (pks PksMonitor) authenticateApi() (string, error) {
//	//defer func() {
//	//	if err != nil {
//	//		pksApiUp.Set(0.0)
//	//	}
//	//}()
//
//	// build api uri to authenticate
//	oauthParams := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=client_credentials&token_format=opaque", pks.UaaCliId, pks.UaaCliSecret)
//	payload := strings.NewReader(oauthParams)
//	method := "POST"
//
//	// create request object
//	req, err := http.NewRequest(method, pks.authApiAddr, payload)
//	if err != nil {
//		pksApiUp.Set(0.0)
//		return "", errors.Wrap(err, "monitor: unable to create new request")
//	}
//
//	req.Headers.Add("Accept", " application/json")
//	req.Headers.Add("Content-Type", "application/x-www-form-urlencoded")
//
//	// making api request
//	res, err := pks.client.Do(req)
//	if err != nil {
//		pksApiUp.Set(0.0)
//		return "", errors.Wrap(err, "monitor: unable to make API request")
//	}
//	defer res.Body.Close()
//
//	body, err := ioutil.ReadAll(res.Body)
//
//	// check success of api call
//	if res.StatusCode != 200 {
//		pksApiUp.Set(0.0)
//		return "", errors.New(fmt.Sprintf("monitor: unable to get API access token: %v - %s", res.Status, body))
//	}
//
//	//unmarshal json resp body to token object
//	var token token
//	err = json.Unmarshal(body, &token)
//	if err != nil {
//		pksApiUp.Set(0.0)
//		return "", errors.WithMessage(err, "monitor: couldn't unmarshal token response")
//	}
//
//	fmt.Printf("authenticated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
//	return token.AccessToken, nil
//}



