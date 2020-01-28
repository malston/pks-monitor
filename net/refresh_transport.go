package net

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

//go:generate counterfeiter . TokenRefresher

type TokenRefresher interface {
	RefreshTokenGrant(clientID, clientSecret, refreshToken string) (string, string, error)
}

//go:generate counterfeiter net/http.RoundTripper

type RefreshTransport struct {
	Transport      http.RoundTripper
	tokenRefresher TokenRefresher
	tokenStore     TokenStore
	clientID       string
	clientSecret   string
}

//go:generate counterfeiter . TokenStore

type TokenStore interface {
	GetAccessToken() string
	GetRefreshToken() string
	SetAccessToken(string)
	SetRefreshToken(string)
}

func NewRefreshTransport(rt http.RoundTripper, tr TokenRefresher, ts TokenStore, clientID, clientSecret string) *RefreshTransport {
	return &RefreshTransport{
		Transport:      rt,
		tokenRefresher: tr,
		tokenStore:     ts,
		clientID:       clientID,
		clientSecret:   clientSecret,
	}
}

func (r *RefreshTransport) RoundTrip(firstReq *http.Request) (*http.Response, error) {
	//secondReq, err := cloneRequest(firstReq)
	//if err != nil {
	//	return nil, err
	//}

	firstReq.Header.Set("Authorization", "Bearer "+r.tokenStore.GetAccessToken())

	response, err := r.Transport.RoundTrip(firstReq)
	if err != nil {
		return response, err
	}
	return response, nil

	//isExpired, err := tokenExpired(response)
	//if err != nil || !isExpired {
	//	return response, err
	//}
	//
	//newAccessToken, newRefreshToken, err := r.tokenRefresher.RefreshTokenGrant(
	//	r.clientID,
	//	r.clientSecret,
	//	r.tokenStore.GetRefreshToken(),
	//)
	//if err != nil {
	//	return nil, err
	//}
	//secondReq.Header.Set("Authorization", "Bearer "+newAccessToken)
	//r.tokenStore.SetAccessToken(newAccessToken)
	//r.tokenStore.SetRefreshToken(newRefreshToken)

	//return r.Transport.RoundTrip(secondReq)
}

func cloneRequest(r *http.Request) (*http.Request, error) {
	if r.Body == nil {
		return r, nil
	}

	r2 := new(http.Request)
	*r2 = *r

	// deep copy the body
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	r2.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

	return r2, nil
}

func tokenExpired(resp *http.Response) (bool, error) {
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
		// Since we fail to decode the error response
		// we cannot ensure that the token is invalid
		return false, nil
	}

	return errResp["error"] == "invalid_token", nil
}
