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
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	testServerOnce   sync.Once
	testServerClient *http.Client
	testServerURL    string
	testServerErr    error
)

// initTestServer starts the real server once for all tests
func initTestServer() (*http.Client, string, error) {
	testServerOnce.Do(func() {
		configFile := "config-test.json"
		config := newConfig(&configFile)
		// Start the real server in a goroutine
		go runServer(config)

		// Wait a bit for the server to start
		time.Sleep(500 * time.Millisecond)

		// Create a client that skips TLS verification (for self-signed cert)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		testServerClient = &http.Client{Transport: tr}
		testServerURL = "https://localhost:8443"
	})
	return testServerClient, testServerURL, testServerErr
}

func TestRootEndpointReturnsIndexHTML(t *testing.T) {
	client, url, err := initTestServer()
	if err != nil {
		t.Fatalf("Failed to initialize test server: %v", err)
	}

	expectedHTML, err := os.ReadFile("content/index.html")
	if err != nil {
		t.Fatalf("%v", err)
	}
	resp, err := client.Get(url + "/")
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
	client, url, err := initTestServer()
	if err != nil {
		t.Fatalf("Failed to initialize test server: %v", err)
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	_, err = client.Get(url + "/")
	if err != nil {
		t.Fatalf("Failed GET request: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"GET / HTTP/1.1" 200`)) {
		t.Errorf("log output does not contain expected value. Got: %s", buf.String())
	}
}

func TestHeadersAreSet(t *testing.T) {
	client, url, err := initTestServer()
	if err != nil {
		t.Fatalf("Failed to initialize test server: %v", err)
	}

	resp, err := client.Get(url + "/")
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
	client, url, err := initTestServer()
	if err != nil {
		t.Fatalf("Failed to initialize test server: %v", err)
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
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
		req, err := http.NewRequest(method, url+"/", nil)
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
	var buf bytes.Buffer
	log.SetOutput(&buf)
	eventsPerSecond := 1
	rl := NewRateLimiter(eventsPerSecond)
	handler := loggingMiddleware(rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	ts := httptest.NewTLSServer(handler)
	defer ts.Close()
	client := ts.Client()
	burstPerSecond := eventsPerSecond * 4 // This value must match the defined in NewRateLimiter.
	for range burstPerSecond {
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
	expectedLog := fmt.Sprintf("\"GET / HTTP/1.1\" %d", http.StatusTooManyRequests)
	if !bytes.Contains(buf.Bytes(), []byte(expectedLog)) {
		t.Errorf("Expected log: %s. Got: %s", expectedLog, buf.String())
	}
}

func TestWikiSubdomainRedirect(t *testing.T) {
	client, url, err := initTestServer()
	if err != nil {
		t.Fatalf("Failed to initialize test server: %v", err)
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Host = "wiki.example.com"
	// Disable redirection to check that the redirection logic is triggered.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected redirect status, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	expectedLocation := "https://example.com/wiki/index.html"
	if location != expectedLocation {
		t.Errorf("Expected Location header to be '%s', got '%s'", expectedLocation, location)
	}
}

func TestHttpRedirection(t *testing.T) {
	client, _, err := initTestServer()
	if err != nil {
		t.Fatalf("Failed to initialize test server: %v", err)
	}

	req, _ := http.NewRequest("GET", "http://localhost:8080/wiki/index.html", nil)
	// Disable redirection to check that the redirection logic is triggered.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMovedPermanently  {
		t.Errorf("Expected status %d, got %d", http.StatusMovedPermanently, resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	expectedProtocol := "https"
	if !strings.HasPrefix(location, expectedProtocol)  {
		t.Errorf("Expected URL to start with '%s', got '%s'", expectedProtocol, resp.Request.URL.String())
	}
}

func TestRequestToWellKnown(t *testing.T) {
	client, _, err := initTestServer()
	if err != nil {
		t.Fatalf("Failed to initialize test server: %v", err)
	}
	resp, err := client.Get("http://localhost:8080/.well-known/1234")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	expectedResp := "foo"
	if !bytes.Equal(bytes.TrimSpace(body), bytes.TrimSpace([]byte(expectedResp))) {
		t.Errorf("Expected response to be '%s', got '%s'", expectedResp, string(body))
	}
}
