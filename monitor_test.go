package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPksMonitor_callApi(t *testing.T) {
	type fields struct {
		apiAddress  string
		accessToken string
		apiUri      string
		respCode    int
		client      *http.Client
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields{
				apiAddress:  "localhost",
				accessToken: "fakeToken",
				respCode:    200,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "not_ok",
			fields: fields{
				apiAddress:  "localhost",
				accessToken: "fakeToken",
				respCode:    500,
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.fields.respCode)
				fmt.Fprintln(w, "hello, client")
			}))

			pks := PksMonitor{
				pksApiAddr:  svr.URL,
				authApiAddr: svr.URL,
				accessToken: tt.fields.accessToken,
				client:      svr.Client(),
			}
			got, err := pks.callApi()
			if (err != nil) != tt.wantErr {
				t.Errorf("callApi() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("callApi() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPksMonitor_callApi_Reauthenticate(t *testing.T) {
	type fields struct {
		apiAddress  string
		accessToken string
		apiUri      string
		token       token
		respCode    int
		client      *http.Client
	}
	tests := []struct {
		name            string
		fields          fields
		wantAccessToken string
		wantErr         bool
	}{
		{
			name: "reauthenticate",
			fields: fields{
				apiAddress:  "localhost",
				accessToken: "fakeToken",
				token: token{
					AccessToken: "faketoken2",
					TokenType:   "type",
					ExpiresIn:   600,
					Scope:       "scope",
					Jti:         "faketoken2",
				},
				respCode: 401,
			},
			wantAccessToken: "faketoken2",
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t := r.Header.Get("Authorization")
				if tt.name == "reauthenticate" {
					if strings.Contains(t, "faketoken2") {
						w.WriteHeader(200)
					} else {
						w.WriteHeader(401)
					}
				} else {
					w.WriteHeader(tt.fields.respCode)
				}
				fmt.Fprintln(w, "hello, client")
			}))
			authSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				jsonToken, _ := json.Marshal(tt.fields.token)
				fmt.Fprintln(w, string(jsonToken))
			}))

			pks := PksMonitor{
				pksApiAddr:  svr.URL,
				authApiAddr: authSvr.URL,
				accessToken: tt.fields.accessToken,
				client:      svr.Client(),
			}
			_, err := pks.callApi()
			gotToken := pks.accessToken
			if (err != nil) != tt.wantErr {
				t.Errorf("callApi() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotToken != tt.wantAccessToken {
				t.Errorf("callApi() got = %v, want %v", gotToken, tt.wantAccessToken)
			}
		})
	}
}

func TestPksMonitor_authenticateApi(t *testing.T) {
	type fields struct {
		apiAddress   string
		apiUri       string
		uaaCliId     string
		uaaCliSecret string
		client       *http.Client
		resp         string
		respCode     int
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields{
				apiAddress:   "localhost",
				uaaCliId:     "fakeId",
				uaaCliSecret: "fakeSecret",
				respCode:     200,
				resp:         `{ "access_token":"fakeToken"}`,
			},
			want:    "fakeToken",
			wantErr: false,
		},
		{
			name: "not_ok",
			fields: fields{
				apiAddress:   "localhost",
				uaaCliId:     "fakeId",
				uaaCliSecret: "fakeSecret",
				resp:         `{ "error":"Bad Credentials"}`,
				respCode:     401,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.fields.respCode)
				fmt.Fprintln(w, tt.fields.resp)
			}))

			pks := PksMonitor{
				pksApiAddr:   svr.URL,
				authApiAddr:  svr.URL,
				client:       svr.Client(),
				uaaCliSecret: tt.fields.uaaCliSecret,
				uaaCliId:     tt.fields.uaaCliId,
			}
			got, err := pks.authenticateApi()
			if (err != nil) != tt.wantErr {
				t.Errorf("callApi() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("callApi() got = %v, want %v", got, tt.want)
			}
			svr.Close()
		})
	}
}
