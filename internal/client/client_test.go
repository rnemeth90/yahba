package client

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
)

func TestNewClient(t *testing.T) {
	cfg := config.Config{
		Timeout:     15,
		KeepAlive:   true,
		Compression: false,
		Logger:      logger.New("error", "stdout", false),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client, got nil")
	}

	if client.Timeout != time.Second*15 {
		t.Fatalf("Expected timeout of 15 seconds, got %v", client.Timeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected http.Transport for client Transport")
	}
	if transport.DisableKeepAlives != !cfg.KeepAlive {
		t.Fatalf("Expected DisableKeepAlives to be %v, got %v", !cfg.KeepAlive, transport.DisableKeepAlives)
	}
}

func TestGETRequest(t *testing.T) {
	// Set up a mock server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	cfg := config.Config{
		URL:    testServer.URL,
		Method: http.MethodGet,
		Logger: logger.New("error", "stdout", false),
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("GET", testServer.URL, nil)
	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send GET request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %v", response.StatusCode)
	}
}

func TestHTTP3Request(t *testing.T) {
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer udpConn.Close()

	server := &http3.Server{
		TLSConfig: http3.ConfigureTLSConfig(&tls.Config{Certificates: []tls.Certificate{selfSignedCert(t)}}),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Proto", r.Proto)
			w.WriteHeader(http.StatusOK)
		}),
	}
	go server.Serve(udpConn)
	defer server.Close()

	cfg := config.Config{
		Timeout:  5,
		HTTP3:    true,
		Insecure: true,
		Logger:   logger.New("error", "stdout", false),
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(client)

	if _, ok := client.Transport.(*http3.Transport); !ok {
		t.Fatalf("Expected http3.Transport, got %T", client.Transport)
	}

	response, err := client.Get("https://" + udpConn.LocalAddr().String())
	if err != nil {
		t.Fatalf("Failed to send HTTP/3 request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %v", response.StatusCode)
	}
	if response.ProtoMajor != 3 || response.Header.Get("X-Proto") != "HTTP/3.0" {
		t.Fatalf("Expected HTTP/3 on both ends, got client %s, server %s", response.Proto, response.Header.Get("X-Proto"))
	}
}

func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// Helper function to identify timeout errors
func isTimeoutError(err error) bool {
	urlErr, ok := err.(*url.Error)
	return ok && urlErr.Timeout()
}
