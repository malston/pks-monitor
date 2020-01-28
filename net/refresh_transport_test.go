package net_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"pks-cli/net"
	"pks-cli/net/netfakes"
)

var _ = Describe("RefreshTransport", func() {
	var (
		mockAPI   *netfakes.FakeRoundTripper
		mockUAA   *netfakes.FakeTokenRefresher
		mockTS    *netfakes.FakeTokenStore
		transport *net.RefreshTransport
	)

	BeforeEach(func() {
		mockAPI = &netfakes.FakeRoundTripper{}
		mockUAA = &netfakes.FakeTokenRefresher{}
		mockTS = &netfakes.FakeTokenStore{}
		transport = net.NewRefreshTransport(
			mockAPI,
			mockUAA,
			mockTS,
			"test-client-id",
			"test-client-secret",
		)
	})

	It("delegates request to the transport it wraps", func() {
		body := ioutil.NopCloser(strings.NewReader("foo"))
		req, err := http.NewRequest("GET", "/something", body)
		Expect(err).ToNot(HaveOccurred())
		expectedErr := errors.New("some-error")
		expectedResp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}
		mockAPI.RoundTripReturns(expectedResp, expectedErr)

		resp, err := transport.RoundTrip(req)

		Expect(resp).To(Equal(expectedResp))
		Expect(err).To(Equal(expectedErr))
	})

	It("adds the access token to the request", func() {
		req, err := http.NewRequest("GET", "/something", nil)
		Expect(err).ToNot(HaveOccurred())
		expectedErr := errors.New("some-error")
		expectedResp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}
		mockAPI.RoundTripReturns(expectedResp, expectedErr)
		mockTS.GetAccessTokenReturns("initial-access-token")

		transport.RoundTrip(req)

		authHeader := mockAPI.RoundTripArgsForCall(0).Header.Get("Authorization")
		Expect(authHeader).To(Equal("Bearer initial-access-token"))
	})

	It("refreshes the token and retries the request if the token is expired", func() {
		req, err := http.NewRequest("GET", "/something", nil)
		Expect(err).ToNot(HaveOccurred())
		expectedResp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "invalid_token"}`)),
		}
		mockAPI.RoundTripReturns(expectedResp, nil)
		mockUAA.RefreshTokenGrantReturns("new-access-token", "new-refresh-token", nil)
		mockTS.GetRefreshTokenReturns("initial-refresh-token")

		transport.RoundTrip(req)

		clientID, clientSecret, refreshToken := mockUAA.RefreshTokenGrantArgsForCall(0)
		Expect(clientID).To(Equal("test-client-id"))
		Expect(clientSecret).To(Equal("test-client-secret"))
		Expect(refreshToken).To(Equal("initial-refresh-token"))
		Expect(mockAPI.RoundTripCallCount()).To(Equal(2))
		authHeader := mockAPI.RoundTripArgsForCall(1).Header.Get("Authorization")
		Expect(authHeader).To(Equal("Bearer new-access-token"))
		Expect(mockTS.SetRefreshTokenArgsForCall(0)).To(Equal("new-refresh-token"))
		Expect(mockTS.SetAccessTokenArgsForCall(0)).To(Equal("new-access-token"))
	})

	It("does not refresh the token if the token has not expired", func() {
		req, err := http.NewRequest("GET", "/something", nil)
		Expect(err).ToNot(HaveOccurred())
		expectedResp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
		}
		mockAPI.RoundTripReturns(expectedResp, nil)

		resp, err := transport.RoundTrip(req)

		Expect(resp).To(Equal(expectedResp))
		Expect(err).To(BeNil())
		Expect(mockAPI.RoundTripCallCount()).To(Equal(1))
	})

	It("does not attempt to refresh the token when an error occurs", func() {
		req, err := http.NewRequest("GET", "/something", nil)
		Expect(err).ToNot(HaveOccurred())
		expectedErr := errors.New("some-error")
		expectedResp := &http.Response{}
		mockAPI.RoundTripReturnsOnCall(0, expectedResp, expectedErr)
		mockAPI.RoundTripReturnsOnCall(1, nil, nil)

		resp, err := transport.RoundTrip(req)

		Expect(mockAPI.RoundTripCallCount()).To(Equal(1))
		Expect(resp).To(Equal(expectedResp))
		Expect(err).To(Equal(expectedErr))
	})

	It("does not retry the request if refresh token grant fails", func() {
		req, err := http.NewRequest("GET", "/something", nil)
		Expect(err).ToNot(HaveOccurred())
		expectedResp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "invalid_token"}`)),
		}
		mockAPI.RoundTripReturns(expectedResp, nil)
		expectedErr := errors.New("some-error")
		mockUAA.RefreshTokenGrantReturns("", "", expectedErr)

		resp, err := transport.RoundTrip(req)

		Expect(mockAPI.RoundTripCallCount()).To(Equal(1))
		Expect(resp).To(BeNil())
		Expect(err).To(Equal(expectedErr))
	})
})
