package main

import (
	"net/http"
	"testing"
)

func TestWikiSubdomainRedirect(t *testing.T) {
	ts := startTestServer("content")
	defer ts.Close()
	client := ts.Client()
	req, _ := http.NewRequest("GET", ts.URL, nil)
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
