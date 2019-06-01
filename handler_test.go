package websocket

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type ExampleMessage struct {
	Body string `json:"body"`
}

func TestHTTPEcho(t *testing.T) {
	echo := func(c *Channel, op OpCode, data []byte) {
		// echo
		c.Send(op, data)
	}

	wh, err := NewHandler(echo, 1, 1)
	if err != nil {
		panic(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wh.UpgradeHandler(w, r)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	err = c.WriteJSON(&ExampleMessage{
		Body: "test",
	})
	if err != nil {
		panic(err)
	}

	var resp ExampleMessage
	err = c.ReadJSON(&resp)
	if err != nil {
		panic(err)
	}

	if resp.Body != "test" {
		t.Error("should have recieved echo response of: test")
	}
}

func TestHTTPSEcho(t *testing.T) {
	echo := func(c *Channel, op OpCode, data []byte) {
		// echo
		c.Send(op, data)
	}

	wh, err := NewHandler(echo, 1, 1)
	if err != nil {
		panic(err)
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wh.UpgradeHandler(w, r)
	}))
	defer server.Close()

	dialer := &websocket.Dialer{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 30 * time.Second,
	}
	dialer.TLSClientConfig = &tls.Config{RootCAs: rootCAs(t, server)}
	url := "wss" + strings.TrimPrefix(server.URL, "https")
	c, _, err := dialer.Dial(url, nil)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	err = c.WriteJSON(&ExampleMessage{
		Body: "test",
	})
	if err != nil {
		panic(err)
	}

	var resp ExampleMessage
	err = c.ReadJSON(&resp)
	if err != nil {
		panic(err)
	}

	if resp.Body != "test" {
		t.Error("should have recieved echo response of: test")
	}
}

func rootCAs(t *testing.T, s *httptest.Server) *x509.CertPool {
	certs := x509.NewCertPool()
	for _, c := range s.TLS.Certificates {
		roots, err := x509.ParseCertificates(c.Certificate[len(c.Certificate)-1])
		if err != nil {
			t.Fatalf("error parsing server's root cert: %v", err)
		}
		for _, root := range roots {
			certs.AddCert(root)
		}
	}
	return certs
}
