// UAA client for Token grants and revocation
package uaa

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Client makes requests to the UAA server at AuthURL
type Client struct {
	AuthURL string
	Client  *http.Client
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Jti          string `json:"jti"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

type responseError struct {
	Name        string `json:"error"`
	Description string `json:"error_description"`
}

// Metadata captures the data returned by the GET /info on a UAA server
// This fields are not exhaustive and can added to over time.
// See: https://docs.cloudfoundry.org/api/uaa/version/4.6.0/index.html#server-information
type Metadata struct {
	Links struct {
		Login string `json:"login"`
	} `json:"links"`
	Prompts struct {
		Passcode []string `json:"passcode"`
	} `json:"prompts"`
}

// PasscodePrompt returns a prompt to tell the user where to get a passcode from.
// If not present in the metadata (PCF installation don't seem to return it), will attempt to
// contruct a plausible URL.
func (md *Metadata) PasscodePrompt() string {
	// Give default in case server doesn't tell us
	if len(md.Prompts.Passcode) == 2 && md.Prompts.Passcode[1] != "" {
		return md.Prompts.Passcode[1]
	}
	var loginURL string
	if md.Links.Login != "" {
		loginURL = md.Links.Login
	} else {
		loginURL = "https://login.system.example.com"
	}
	return fmt.Sprintf("One Time Code ( Get one at %s/passcode )", loginURL)
}

func (e *responseError) Error() string {
	if e.Description == "" {
		return e.Name
	}

	return fmt.Sprintf("%s %s", e.Name, e.Description)
}

// ClientCredentialGrant requests a Token using client_credentials grant type
func (u *Client) ClientCredentialGrant(clientId, clientSecret string) (Token, error) {
	values := url.Values{
		"grant_type":    {"client_credentials"},
		"response_type": {"Token"},
		"client_id":     {clientId},
		"client_secret": {clientSecret},
	}

	token, err := u.tokenGrantRequest(values)

	return token, err
}

// RefreshTokenGrant requests a new access Token and refresh Token using refresh_token grant type
func (u *Client) RefreshTokenGrant(clientId, clientSecret, refreshToken string) (string, string, error) {
	values := url.Values{
		"grant_type":    {"refresh_token"},
		"response_type": {"Token"},
		"client_id":     {clientId},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
	}

	token, err := u.tokenGrantRequest(values)

	return token.AccessToken, token.RefreshToken, err
}

func (u *Client) tokenGrantRequest(headers url.Values) (Token, error) {
	var t Token

	request, err := http.NewRequest("POST", u.AuthURL+"/oauth/token", bytes.NewBufferString(headers.Encode()))
	if err != nil {
		return t, err
	}

	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	csrfUrl, _ := url.Parse(u.AuthURL)
	if u.Client.Jar.Cookies(csrfUrl) != nil {
		csrfVal := strings.Split(u.Client.Jar.Cookies(csrfUrl)[0].String(), "=")[1]
		request.Header.Add("X-CSRF-TOKEN", csrfVal)
	}

	response, err := u.Client.Do(request)

	if err != nil {
		return t, err
	}

	defer response.Body.Close()
	defer io.Copy(ioutil.Discard, response.Body)

	decoder := json.NewDecoder(response.Body)

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		err = decoder.Decode(&t)
		fmt.Printf("code: %s, token: %+v\n", response.StatusCode, t)
		return t, err
	}

	respErr := responseError{}

	if err := decoder.Decode(&respErr); err != nil {
		fmt.Printf("code: %s, err: %+v\n", response.StatusCode, err)
		return t, err
	}

	return t, &respErr
}

// RevokeToken revokes the given access Token
func (u *Client) RevokeToken(accessToken string) error {
	segments := strings.Split(accessToken, ".")

	if len(segments) < 2 {
		return errors.New("access Token missing segments")
	}

	jsonPayload, err := base64.RawURLEncoding.DecodeString(segments[1])

	if err != nil {
		return errors.New("could not base64 decode Token payload")
	}

	payload := make(map[string]interface{})
	json.Unmarshal(jsonPayload, &payload)
	jti, ok := payload["jti"].(string)

	if !ok {
		return errors.New("could not parse jti from payload")
	}

	request, err := http.NewRequest(http.MethodDelete, u.AuthURL+"/oauth/Token/revoke/"+jti, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := u.Client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Received HTTP %d error while revoking Token from auth server: %q", resp.StatusCode, body)
	}

	return nil
}
