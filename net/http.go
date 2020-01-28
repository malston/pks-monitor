package net

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/pkg/errors"

	"net/http"
	"time"
)

// HTTPClient returns an http.Client that has TLS and proxy configured.
func HTTPClient(insecure bool, cert []byte) (*http.Client, error) {
	transport := Transport(true, nil)
	if !insecure {
		certPool, err := CertPool(cert)
		if err != nil {
			return nil, errors.Wrap(err, "net: could not create Cert Pool")
		}
		transport = Transport(false, certPool)
	}
	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
		//Jar:       nil,
	}, nil
}

func Transport(insecure bool, certPool *x509.CertPool) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: insecure,
	}

	if !insecure {
		transport.TLSClientConfig.RootCAs = certPool
	}
	return transport
}

// CertPool returns an x509.CertPool for the provided cert.
func CertPool(cert []byte) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(cert)
	if !ok {
		return nil, errors.New("failed to load ca cert")
	}
	return certPool, nil
}

