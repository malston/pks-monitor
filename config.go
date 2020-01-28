package monitor

import (
	"github.com/pupimvictor/pks-monitor/uaa"
	"net"
	pksNet "github.com/pupimvictor/pks-monitor/net"
	"net/http"
	"net/url"
	"strings"
)

// Config represents the configuration for the PKS CLI. This includes things
// like auth tokens, flags, and the PKS API host.
type Config struct {
	API                 string `yaml:"api"`
	CACert              string `yaml:"ca_cert"`
	Username            string `yaml:"username"`
	SkipSSLVerification bool   `yaml:"skip_ssl_verification"`
	AccessToken         string `yaml:"access_token"`
	RefreshToken        string `yaml:"refresh_token"`

	Headers *http.Header

	UaaCliId     string
	UaaCliSecret string
}

var (
	// APIPort is the port used to communicate with the PKS API.
	APIPort = "9021"
	// UAAPort is the port used to communicate with the PKS UAA.
	UAAPort = "8443"
)

// GetAccessToken returns the access token.
func (c *Config) GetAccessToken() string {
	return c.AccessToken
}

// GetRefreshToken returns the refresh token.
func (c *Config) GetRefreshToken() string {
	return c.RefreshToken
}

// SetAccessToken sets the access token.
func (c *Config) SetAccessToken(at string) {
	c.AccessToken = at
}

// SetRefreshToken sets the refresh token.
func (c *Config) SetRefreshToken(rt string) {
	c.RefreshToken = rt
}

func CreateHttpClient (c *Config) (*http.Client, error) {
	uaaHTTPClient, err := pksNet.HTTPClient(c.SkipSSLVerification, []byte(c.CACert))
	if err != nil {
		return nil, err
	}
	apiHTTPClient, err := pksNet.HTTPClient(c.SkipSSLVerification, []byte(c.CACert))
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(c.API)
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

	uaaClient := &uaa.Client{
		AuthURL:    u.Scheme + "://" + host + ":" + UAAPort,
		Client: uaaHTTPClient,
	}

	apiHTTPClient.Transport = pksNet.NewRefreshTransport(
		apiHTTPClient.Transport,
		uaaClient,
		c,
		c.UaaCliId,
		c.UaaCliSecret,
	)
	return apiHTTPClient, nil
}
