package monitor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPksMonitor_callApi(t *testing.T) {
	type fields struct {
		apiAddress   string
		accessToken  string
		apiUri string
		respCode int
		client       *http.Client
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields {
				apiAddress:"localhost",
				accessToken: "fakeToken",
				respCode: 200,
			},
			want: true,
			wantErr: false,
		},
		{
			name: "not_ok",
			fields: fields {
				apiAddress:"localhost",
				accessToken: "fakeToken",
				respCode: 500,
			},
			want: false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.fields.respCode)
				fmt.Fprintln(w, "Hello, client")
			}))

			pks := PksMonitor{
				apiAddress:   svr.URL,
				accessToken:  tt.fields.accessToken,
				client: svr.Client(),
			}
			pksApiClusters = ""
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

func TestPksMonitor_authenticateApi(t *testing.T) {
	type fields struct {
		apiAddress   string
		apiUri string
		uaaCliId string
		uaaCliSecret string
		client       *http.Client
		resp string
		respCode int
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields {
				apiAddress:"localhost",
				uaaCliId: "fakeId",
				uaaCliSecret: "fakeSecret",
				respCode: 200,
				resp: `{ "access_token":"fakeToken"}`,
			},
			want: "fakeToken",
			wantErr: false,
		},
		{
			name: "not_ok",
			fields: fields {
				apiAddress:"localhost",
				uaaCliId: "fakeId",
				uaaCliSecret: "fakeSecret",
				resp: `{ "error":"Bad Credentials"}`,
				respCode: 401,
			},
			want: "",
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
				apiAddress:   svr.URL,
				client: svr.Client(),
				uaaCliSecret: tt.fields.uaaCliSecret,
				uaaCliId: tt.fields.uaaCliId,
			}
			pksApiAuth = ""
			got, err := pks.authenticateApi()
			if (err != nil) != tt.wantErr {
				t.Errorf("callApi() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != nil) && got.AccessToken != tt.want {
				t.Errorf("callApi() got = %v, want %v", got, tt.want)
			}
			svr.Close()
		})
	}
}
