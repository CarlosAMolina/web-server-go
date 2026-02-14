package main

import (
	"bytes"
	"io"
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

func startTestServer(contentDir string) *httptest.Server {
	fs := http.FileServer(http.Dir(contentDir))
	handler := loggingMiddleware(http.StripPrefix("/", fs))
	// NewTLSServer's client trusts its self-signed certificate.
	ts := httptest.NewTLSServer(handler)
	return ts
}
