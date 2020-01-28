package net_test

import (
	"net/http"
	"net/url"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"pks-cli/net"
)

var _ = Describe("HTTPClient", func() {
	Context("for insecure connection", func() {
		It("uses the transport from net.Transport", func() {
			client, err := net.HTTPClient(true, nil)

			Expect(err).ToNot(HaveOccurred())
			t := client.Transport.(*http.Transport)
			Expect(funcAddr(t.Proxy)).To(Equal(funcAddr(http.ProxyFromEnvironment)))
			Expect(t.TLSClientConfig.InsecureSkipVerify).To(BeTrue())
		})

		It("sets timeout for the client", func() {
			client, err := net.HTTPClient(true, nil)

			Expect(err).ToNot(HaveOccurred())
			Expect(client.Timeout).To(Equal(60 * time.Second))
		})
	})

	Context("for secure connection", func() {
		var client *http.Client
		BeforeEach(func() {
			cert := []byte(`-----BEGIN CERTIFICATE-----
MIICEzCCAXygAwIBAgIQMIMChMLGrR+QvmQvpwAU6zANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB
iQKBgQDuLnQAI3mDgey3VBzWnB2L39JUU4txjeVE6myuDqkM/uGlfjb9SjY1bIw4
iA5sBBZzHi3z0h1YV8QPuxEbi4nW91IJm2gsvvZhIrCHS3l6afab4pZBl2+XsDul
rKBxKKtD1rGxlG4LjncdabFn9gvLZad2bSysqz/qTAUStTvqJQIDAQABo2gwZjAO
BgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/BAUw
AwEB/zAuBgNVHREEJzAlggtleGFtcGxlLmNvbYcEfwAAAYcQAAAAAAAAAAAAAAAA
AAAAATANBgkqhkiG9w0BAQsFAAOBgQCEcetwO59EWk7WiJsG4x8SY+UIAA+flUI9
tyC4lNhbcF2Idq9greZwbYCqTTTr2XiRNSMLCOjKyI7ukPoPjo16ocHj+P3vZGfs
h1fIw3cSS2OolhloGw/XM6RWPWtPAlGykKLciQrBru5NAPvCMsb/I1DAceTiotQM
fblo6RBxUQ==
-----END CERTIFICATE-----`)
			var err error
			client, err = net.HTTPClient(false, cert)
			Expect(err).ToNot(HaveOccurred())
		})

		It("uses the transport from net.Transport", func() {
			t := client.Transport.(*http.Transport)
			Expect(funcAddr(t.Proxy)).To(Equal(funcAddr(http.ProxyFromEnvironment)))
			Expect(t.TLSClientConfig.InsecureSkipVerify).To(BeFalse())
		})

		It("sets timeout for the client", func() {
			Expect(client.Timeout).To(Equal(60 * time.Second))
		})
	})
})

var _ = Describe("Transport", func() {
	Context("all cases", func() {
		It("uses default http transport values", func() {
			t := net.Transport(true, nil)
			Expect(t.IdleConnTimeout).To(Equal(90 * time.Second))
			Expect(t.TLSHandshakeTimeout).To(Equal(10 * time.Second))
			Expect(t.ExpectContinueTimeout).To(Equal(1 * time.Second))
			Expect(t.MaxIdleConns).To(Equal(100))
		})
	})

	Context("secure", func() {
		It("returns a transport with a valid certificate", func() {
			certPool, err := net.CertPool([]byte(`-----BEGIN CERTIFICATE-----
MIICEzCCAXygAwIBAgIQMIMChMLGrR+QvmQvpwAU6zANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB
iQKBgQDuLnQAI3mDgey3VBzWnB2L39JUU4txjeVE6myuDqkM/uGlfjb9SjY1bIw4
iA5sBBZzHi3z0h1YV8QPuxEbi4nW91IJm2gsvvZhIrCHS3l6afab4pZBl2+XsDul
rKBxKKtD1rGxlG4LjncdabFn9gvLZad2bSysqz/qTAUStTvqJQIDAQABo2gwZjAO
BgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/BAUw
AwEB/zAuBgNVHREEJzAlggtleGFtcGxlLmNvbYcEfwAAAYcQAAAAAAAAAAAAAAAA
AAAAATANBgkqhkiG9w0BAQsFAAOBgQCEcetwO59EWk7WiJsG4x8SY+UIAA+flUI9
tyC4lNhbcF2Idq9greZwbYCqTTTr2XiRNSMLCOjKyI7ukPoPjo16ocHj+P3vZGfs
h1fIw3cSS2OolhloGw/XM6RWPWtPAlGykKLciQrBru5NAPvCMsb/I1DAceTiotQM
fblo6RBxUQ==
-----END CERTIFICATE-----`))
			Expect(err).ToNot(HaveOccurred())

			t := net.Transport(false, certPool)
			Expect(funcAddr(t.Proxy)).To(Equal(funcAddr(http.ProxyFromEnvironment)))
			Expect(t.TLSClientConfig.InsecureSkipVerify).To(BeFalse())
			Expect(t.TLSClientConfig.RootCAs).ToNot(BeZero())
		})

		It("returns a error with an invalid certificate", func() {
			_, err := net.CertPool(nil)
			Expect(err).To(MatchError("failed to load ca cert"))
		})
	})

	Context("insecure", func() {
		It("returns a transport that has tls and proxy configured", func() {
			t := net.Transport(true, nil)
			Expect(funcAddr(t.Proxy)).To(Equal(funcAddr(http.ProxyFromEnvironment)))
			Expect(t.TLSClientConfig.InsecureSkipVerify).To(BeTrue())
		})
	})
})

func funcAddr(f func(*http.Request) (*url.URL, error)) uintptr {
	return reflect.ValueOf(f).Pointer()
}
