package net

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

//go:generate counterfeiter net/http.RoundTripper

type AuthTransport struct {
	Transport      http.RoundTripper
	tokenStore     TokenStore

	//todo check if cliId and secret are necessary here
	clientID       string
	clientSecret   string
}

//go:generate counterfeiter . TokenStore

type TokenStore interface {
	GetAccessToken() string
	SetAccessToken(string)
}

func NewAuthTransport(rt http.RoundTripper, ts TokenStore, clientID, clientSecret string) *AuthTransport {
	return &AuthTransport{
		Transport:      rt,
		tokenStore:     ts,
		clientID:       clientID,
		clientSecret:   clientSecret,
	}
}

func (r *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer " + r.tokenStore.GetAccessToken())

	response, err := r.Transport.RoundTrip(req)
	if err != nil {
		return response, err
	}
	return response, nil
}


func TokenExpired(resp *http.Response) (bool, error) {
	if resp.StatusCode < 400 {
		return false, nil
	}

	var errResp map[string]string
	buf, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return false, err
	}

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

	decoder := json.NewDecoder(bytes.NewBuffer(buf))
	err = decoder.Decode(&errResp)
	if err != nil {
		return true, err
	}

	return errResp["error"] == "invalid_token", nil
}
