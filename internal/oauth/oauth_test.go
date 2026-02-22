package oauth

import (
	"context"
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

func TestBuildProxyAuthorizeURL(t *testing.T) {
	authURL := buildProxyAuthorizeURL("https://proxy.example.com", 12345, "csrf-abc")

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse proxy auth URL: %v", err)
	}

	if u.Path != "/authorize" {
		t.Errorf("path = %q, want /authorize", u.Path)
	}
	if u.Query().Get("port") != "12345" {
		t.Errorf("port = %q, want 12345", u.Query().Get("port"))
	}
	if u.Query().Get("state") != "csrf-abc" {
		t.Errorf("state = %q, want csrf-abc", u.Query().Get("state"))
	}
}

func TestBuildProxyAuthorizeURLTrailingSlash(t *testing.T) {
	authURL := buildProxyAuthorizeURL("https://proxy.example.com/", 8080, "s")
	if !strings.Contains(authURL, "proxy.example.com/authorize?") {
		t.Errorf("trailing slash not trimmed: %s", authURL)
	}
}

func TestHandleProxyDone(t *testing.T) {
	ch := make(chan flowResult, 1)
	handler := handleProxyDone(ch, "expected-state")

	// Build a POST request with form data.
	form := url.Values{
		"token":     {"xoxp-test-token"},
		"app_token": {"xapp-test-app-token"},
		"user_id":   {"U12345"},
		"team_id":   {"T12345"},
		"team_name": {"Test Team"},
		"state":     {"expected-state"},
	}
	req := httptest.NewRequest(http.MethodPost, "/done", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("unexpected error: %v", res.err)
		}
		if res.result.UserToken != "xoxp-test-token" {
			t.Errorf("UserToken = %q, want %q", res.result.UserToken, "xoxp-test-token")
		}
		if res.result.AppToken != "xapp-test-app-token" {
			t.Errorf("AppToken = %q, want %q", res.result.AppToken, "xapp-test-app-token")
		}
		if res.result.TeamID != "T12345" {
			t.Errorf("TeamID = %q", res.result.TeamID)
		}
		if res.result.TeamName != "Test Team" {
			t.Errorf("TeamName = %q", res.result.TeamName)
		}
		if res.result.UserID != "U12345" {
			t.Errorf("UserID = %q", res.result.UserID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestHandleProxyDoneStateMismatch(t *testing.T) {
	ch := make(chan flowResult, 1)
	handler := handleProxyDone(ch, "correct-state")

	form := url.Values{
		"token": {"xoxp-token"},
		"state": {"wrong-state"},
	}
	req := httptest.NewRequest(http.MethodPost, "/done", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for state mismatch, got %d", w.Code)
	}

	select {
	case res := <-ch:
		if res.err == nil || !strings.Contains(res.err.Error(), "state mismatch") {
			t.Errorf("expected state mismatch error, got: %v", res.err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestHandleProxyDoneMethodNotAllowed(t *testing.T) {
	ch := make(chan flowResult, 1)
	handler := handleProxyDone(ch, "state")

	req := httptest.NewRequest(http.MethodGet, "/done", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleProxyDoneMissingToken(t *testing.T) {
	ch := make(chan flowResult, 1)
	handler := handleProxyDone(ch, "state")

	form := url.Values{
		"state": {"state"},
		// no token
	}
	req := httptest.NewRequest(http.MethodPost, "/done", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	select {
	case res := <-ch:
		if res.err == nil || !strings.Contains(res.err.Error(), "no token") {
			t.Errorf("expected 'no token' error, got: %v", res.err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestDirectCallbackStateMismatch(t *testing.T) {
	ch := make(chan flowResult, 1)
	handler := handleDirectCallback(ch, "correct-state", "id", "secret", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodGet, "/callback?code=abc&state=wrong-state", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRunTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := Run(ctx, Params{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		OpenBrowser:  func(string) error { return nil },
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestRunRequiresSecretOrProxy(t *testing.T) {
	ctx := context.Background()
	_, err := Run(ctx, Params{
		ClientID:    "test-id",
		OpenBrowser: func(string) error { return nil },
	})
	if err == nil {
		t.Fatal("expected error when neither secret nor proxy is set")
	}
	if !strings.Contains(err.Error(), "client_secret") {
		t.Errorf("expected error about client_secret/proxy_url, got: %v", err)
	}
}
