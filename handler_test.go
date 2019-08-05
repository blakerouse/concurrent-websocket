package websocket

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type ExampleMessage struct {
	Count int `json:"count"`
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

	for i := 0; i < 3; i++ {
		err = c.WriteJSON(&ExampleMessage{
			Count: i,
		})
		if err != nil {
			panic(err)
		}

		var resp ExampleMessage
		err = c.ReadJSON(&resp)
		if err != nil {
			panic(err)
		}

		if resp.Count != i {
			t.Errorf("should have recieved echo response of: count = %d", i)
		}
	}
}

func TestHTTPSEcho(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	echo := func(c *Channel, op OpCode, data []byte) {
		c.SetOnClose(func() {
			wg.Done()
		})

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

	for i := 0; i < 3; i++ {
		err = c.WriteJSON(&ExampleMessage{
			Count: i,
		})
		if err != nil {
			c.Close()
			panic(err)
		}

		var resp ExampleMessage
		err = c.ReadJSON(&resp)
		if err != nil {
			c.Close()
			panic(err)
		}

		if resp.Count != i {
			t.Errorf("should have recieved echo response of: count = %d", i)
		}
	}
	c.Close()
	timedOut := waitTimeout(&wg, time.Second)
	if timedOut {
		t.Errorf("timed out waiting for OnClose to be called")
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

// waitGroupTimeout waits for the waitgroup for the specified max timeout.
func waitGroupTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
