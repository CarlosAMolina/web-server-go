package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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

func startTestServer(contentDir string) *httptest.Server {
	fs := http.FileServer(http.Dir(contentDir))
	handler := loggingMiddleware(http.StripPrefix("/", fs))
	// NewTLSServer's client trusts its self-signed certificate.
	ts := httptest.NewTLSServer(handler)
	return ts
}
