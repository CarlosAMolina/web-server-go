package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
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

func TestHeadersAreSet(t *testing.T) {
	ts := startTestServer("content")
	defer ts.Close()
	client := ts.Client()
	resp, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("Failed GET request: %v", err)
	}
	defer resp.Body.Close()
	headers := make(map[string]string)
	headers["Content-Security-Policy"] = "default-src 'self'"
	headers["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains"
	headers["X-Content-Type-Options"] = "nosniff"
	headers["X-Frame-Options"] = "SAMEORIGIN"
	for header, expectedValue := range headers {
		result := resp.Header.Get(header)
		if result != expectedValue {
			t.Errorf("Expected %s header to be '%s', but got '%s'", header, expectedValue, result)
		}
	}
}

func TestOnlyGETMethodAllowed(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	ts := startTestServer("content")
	defer ts.Close()
	client := ts.Client()
	methods := []string{
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace,
	}
	for _, method := range methods {
		req, err := http.NewRequest(method, ts.URL+"/", nil)
		if err != nil {
			t.Fatalf("Failed to create %s request: %v", method, err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed %s request: %v", method, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d for method %s, got %d", http.StatusMethodNotAllowed, method, resp.StatusCode)
		}
		header := resp.Header.Get("Allow")
		expectedHeader := "GET"
		if header != expectedHeader {
			t.Errorf("Expected Allow header to be '%s', but got '%s'", expectedHeader, header)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		expectedBody := "Method Not Allowed\n"
		if method == http.MethodHead {
			expectedBody = ""
		}
		if string(body) != expectedBody {
			t.Errorf("Expected body %q for method %s, got %q", expectedBody, method, string(body))
		}
		expectedLog := fmt.Sprintf("\"%s / HTTP/1.1\" %d", method, resp.StatusCode)
		if !bytes.Contains(buf.Bytes(), []byte(expectedLog)) {
			t.Errorf("Expected log: %s. Got: %s", expectedLog, buf.String())
		}
		buf.Reset()
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

func TestRateLimiter(t *testing.T) {
	handler := rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	ts := httptest.NewTLSServer(handler)
	defer ts.Close()
	client := ts.Client()
	for range 3 {
		resp, err := client.Get(ts.URL)
		if err != nil {
			t.Fatalf("Failed GET request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
	}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("Failed GET request: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, resp.StatusCode)
	}
}

// TODO use the main.go server config to work with expected timoeouts
func startTestServer(contentDir string) *httptest.Server {
	fs := http.FileServer(http.Dir(contentDir))
	handler := loggingMiddleware(requestMiddleware(http.StripPrefix("/", fs)))
	// NewTLSServer's client trusts its self-signed certificate.
	ts := httptest.NewTLSServer(handler)
	return ts
}
