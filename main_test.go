package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestRootEndpointReturnsIndexHTML(t *testing.T) {
	expectedHTML, err := os.ReadFile("content/index.html")
	if err != nil {
		t.Fatalf("%v", err)
	}
	ts := startTestServer("content")
	defer ts.Close()
	client := ts.Client()
	resp, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("Failed GET request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(expectedHTML), bytes.TrimSpace(body)) {
		t.Errorf("Response body != index.html content.\n\nExpected (trimmed):\n%s\n\nGot (trimmed):\n%s",
			bytes.TrimSpace(expectedHTML),
			bytes.TrimSpace(body))
	}
}

func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	ts := startTestServer("content")
	defer ts.Close()
	client := ts.Client()
	_, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("Failed GET request: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"GET / HTTP/1.1" 200`)) {
		t.Errorf("log output does not contain expected value. Got: %s", buf.String())
	}
}

func TestConnectionIsClosed(t *testing.T) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	ts.Config.ReadTimeout = 30 * time.Millisecond
	ts.Config.WriteTimeout = 30 * time.Millisecond
	ts.Config.IdleTimeout = 30 * time.Millisecond
	ts.StartTLS()
	defer ts.Close()
	// Open a TLS connection.
	conn, err := tls.Dial("tcp", ts.Listener.Addr().String(), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}
	defer conn.Close()
	// The last \r\n is required to avoid a EOF closed connection.
	connectionRequest := "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"
	// Send a request over the connection.
	_, err = conn.Write([]byte(connectionRequest))
	if err != nil {
		t.Fatalf("Connection not established: %v", err)
	}
	// Read the entire response from the first request. This makes the connection
	// idle, so that the IdleTimeout can trigger.
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("The first request was not successful: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	// Wait a time higher than the timeout.
	time.Sleep(200 * time.Millisecond)
	// Send a request over the connection.
	_, err = conn.Write([]byte(connectionRequest))
	if err != nil {
		// Note. A write to a closed connection may fail, so this panic may be omitted.
		panic(err)
	}
	// Read server response
	one := make([]byte, 1)
	_, err = conn.Read(one)
	if err != io.EOF {
		t.Errorf("The server should have closed the connection. Expected EOF error, but got: %v", err)
	}
}

// TODO use the main.go server config to work with expected timoeouts
func startTestServer(contentDir string) *httptest.Server {
	fs := http.FileServer(http.Dir(contentDir))
	handler := loggingMiddleware(http.StripPrefix("/", fs))
	// NewTLSServer's client trusts its self-signed certificate.
	ts := httptest.NewTLSServer(handler)
	return ts
}
