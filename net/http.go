package net

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"golang.org/x/net/publicsuffix"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"
)

// HTTPClient returns an http.Client that has TLS and proxy configured.
func HTTPClient(insecure bool, cert []byte) (*http.Client, error) {
	transport := Transport(true, nil)
	if !insecure {
		certPool, err := CertPool(cert)
		if err != nil {
			return nil, err
		}
		transport = Transport(false, certPool)
	}
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("http: couldn't create cookie jar: %+v\n", err)
	}
	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
		Jar: jar,
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

// ParseCert a helper method to parse a string as a cert or path to a cert
// and validate that it is a correctly formatted cert.
func ParseCert(caCertPath string) ([]byte, error) {
	var pemBytes []byte
	if caCertPath == "" {
		return nil, errors.New("CA Cert path missing")
	}
	_, err := os.Stat(caCertPath)
	believeInputIsFilePath := err == nil
	if err != nil {
		pemBytes = []byte(caCertPath)
	} else {
		pemBytes, err = ioutil.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read file: %s", caCertPath)
		}
	}
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(pemBytes)
	if !ok {
		if believeInputIsFilePath {
			return nil, fmt.Errorf("Failed to parse certificate from file: %s", caCertPath)
		}
		return nil, fmt.Errorf("Failed to parse certificate from command line (or file does not exist): %s", caCertPath)
	}

	return pemBytes, nil
}
