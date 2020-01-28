package monitor

import (
	"github.com/pkg/errors"
	pksNet "github.com/pupimvictor/pks-monitor/net"
	"github.com/pupimvictor/pks-monitor/uaa"
	"net/http"
	"net/url"
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

// SetAccessToken sets the access token.
func (c *Config) SetAccessToken(at string) {
	c.AccessToken = at
}

func CreateHttpClient(c *Config) (*http.Client, error) {
	apiHTTPClient, err := pksNet.HTTPClient(c.SkipSSLVerification, []byte(c.CACert))
	if err != nil {
		return nil, errors.Wrap(err, "monitor: could not create HTTPClient")
	}
	apiHTTPClient.Transport = pksNet.NewAuthTransport(
		apiHTTPClient.Transport,
		c,
		c.UaaCliId,
		c.UaaCliSecret,
	)
	return apiHTTPClient, nil
}

func CreateUaaClient(c *Config) (*uaa.Client, error) {
	uaaHTTPClient, err := pksNet.HTTPClient(c.SkipSSLVerification, []byte(c.CACert))
	if err != nil {
		return nil, errors.Wrap(err, "monitor: could not create HTTPClient")
	}
	u, err := url.Parse(c.API)
	if err != nil {
		return nil, err
	}
	u.Host = u.Host + ":" + UAAPort
	uaaClient := &uaa.Client{
		AuthURL: *u,
		Client:  uaaHTTPClient,
	}
	return uaaClient, nil
}
