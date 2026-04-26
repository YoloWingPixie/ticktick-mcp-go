package safety

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitedTransport(t *testing.T) {
	calls := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	// 1 request per second, burst of 1
	transport := NewRateLimitedTransport(http.DefaultTransport, 1, 1)
	client := &http.Client{Transport: transport}

	// First request should be immediate
	start := time.Now()
	resp, err := client.Get(backend.URL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	resp.Body.Close()
	firstDuration := time.Since(start)

	if firstDuration > 200*time.Millisecond {
		t.Errorf("first request took %v, expected near-instant", firstDuration)
	}

	// Second request should be delayed by ~1s
	start = time.Now()
	resp, err = client.Get(backend.URL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	resp.Body.Close()
	secondDuration := time.Since(start)

	if secondDuration < 500*time.Millisecond {
		t.Errorf("second request took %v, expected at least ~1s delay", secondDuration)
	}

	if calls != 2 {
		t.Errorf("expected 2 backend calls, got %d", calls)
	}
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient(nil, 30*time.Second, DefaultRate, DefaultBurst)

	if client.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", client.Timeout)
	}

	if client.CheckRedirect == nil {
		t.Fatal("CheckRedirect should not be nil")
	}

	if client.Transport == nil {
		t.Fatal("Transport should not be nil")
	}
}

func TestCheckRedirect_SameHost(t *testing.T) {
	// Set up a server that redirects to itself
	var redirectURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, redirectURL+"/target", http.StatusFound)
			return
		}
		// Check that Authorization header is still present
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	redirectURL = ts.URL

	client := NewHTTPClient(nil, 5*time.Second, 100, 100)

	req, err := http.NewRequest("GET", ts.URL+"/redirect", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (Authorization should be preserved on same-host redirect)", resp.StatusCode)
	}
}

func TestCheckRedirect_DifferentHost(t *testing.T) {
	// Target server checks that Authorization is NOT present
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Authorization header should have been stripped"))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	// Source server redirects to a different host (the target)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/target", http.StatusFound)
	}))
	defer source.Close()

	client := NewHTTPClient(nil, 5*time.Second, 100, 100)

	req, err := http.NewRequest("GET", source.URL+"/start", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (Authorization should be stripped on cross-host redirect)", resp.StatusCode)
	}
}
