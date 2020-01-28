// UAA client for Token grants and revocation
package uaa

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Client makes requests to the UAA server at AuthURL
type Client struct {
	AuthURL url.URL
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

func (u *Client) tokenGrantRequest(headers url.Values) (Token, error) {
	var t Token

	request, err := http.NewRequest("POST", u.AuthURL.String() + "/oauth/token", bytes.NewBufferString(headers.Encode()))
	if err != nil {
		return t, errors.Wrap(err, "uaa: unable to create tokenGrantRequest")
	}

	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	response, err := u.Client.Do(request)

	if err != nil {
		return t, err
	}

	defer response.Body.Close()
	defer io.Copy(ioutil.Discard, response.Body)

	decoder := json.NewDecoder(response.Body)

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		err = decoder.Decode(&t)
		return t, errors.Wrap(err, "uaa: unable to decode token")
	}

	respErr := responseError{}

	if err := decoder.Decode(&respErr); err != nil {
		return t, errors.Wrapf(err, "code: %s\n", response.StatusCode)
	}

	return t, &respErr
}
