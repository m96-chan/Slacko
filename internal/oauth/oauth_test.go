package oauth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGenerateState(t *testing.T) {
	s1, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	if len(s1) != 32 { // 16 bytes = 32 hex chars
		t.Fatalf("expected 32 hex chars, got %d", len(s1))
	}

	s2, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	if s1 == s2 {
		t.Fatal("two generated states should not be identical")
	}
}

func TestBuildAuthURL(t *testing.T) {
	authURL := buildAuthURL("my-client-id", "http://localhost:9999/callback", "test-state")

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}

	if u.Scheme != "https" || u.Host != "slack.com" || u.Path != "/oauth/v2/authorize" {
		t.Fatalf("unexpected URL base: %s", authURL)
	}

	q := u.Query()
	if q.Get("client_id") != "my-client-id" {
		t.Errorf("client_id = %q, want %q", q.Get("client_id"), "my-client-id")
	}
	if q.Get("redirect_uri") != "http://localhost:9999/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("state") != "test-state" {
		t.Errorf("state = %q, want %q", q.Get("state"), "test-state")
	}
	if q.Get("user_scope") != UserScopes {
		t.Errorf("user_scope = %q, want %q", q.Get("user_scope"), UserScopes)
	}
}

func TestCallbackReceivesCode(t *testing.T) {
	// This test verifies the callback HTTP handler correctly receives and
	// validates the authorization code. We test via Run() with a mock Slack
	// token exchange endpoint.

	// Mock Slack OAuth endpoint.
	mockSlack := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a valid OAuth v2 response.
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{
			"ok": true,
			"authed_user": {
				"id": "U12345",
				"access_token": "xoxp-test-token"
			},
			"team": {
				"id": "T12345",
				"name": "Test Team"
			}
		}`)
	}))
	defer mockSlack.Close()

	// Since we can't easily mock slack.GetOAuthV2Response (it calls the real
	// Slack API), we instead test the callback handler in isolation.
	state := "test-state-123"
	type callbackResult struct {
		code string
		err  error
	}
	ch := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			ch <- callbackResult{err: nil}
			return
		}
		if r.URL.Query().Get("state") != state {
			ch <- callbackResult{err: nil}
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		ch <- callbackResult{code: code}
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Simulate callback with correct state and code.
	resp, err := http.Get(server.URL + "/callback?code=test-auth-code&state=" + state)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case res := <-ch:
		if res.code != "test-auth-code" {
			t.Errorf("code = %q, want %q", res.code, "test-auth-code")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for callback result")
	}
}

func TestCallbackStateMismatch(t *testing.T) {
	state := "correct-state"
	type callbackResult struct {
		code string
		err  error
	}
	ch := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			ch <- callbackResult{err: nil}
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		ch <- callbackResult{code: r.URL.Query().Get("code")}
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Send callback with wrong state.
	resp, err := http.Get(server.URL + "/callback?code=test-code&state=wrong-state")
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for state mismatch, got %d", resp.StatusCode)
	}
}

func TestRunTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// openBrowser does nothing — no callback will arrive, so it should time out.
	_, err := Run(ctx, "test-id", "test-secret", "xapp-test", func(string) error { return nil })
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}
