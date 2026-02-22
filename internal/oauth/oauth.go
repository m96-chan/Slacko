package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

// UserScopes is the set of OAuth user scopes required by Slacko.
const UserScopes = "channels:history,channels:read,channels:write,chat:write," +
	"emoji:read,files:read,files:write," +
	"groups:history,groups:read,groups:write," +
	"im:history,im:read,im:write," +
	"mpim:history,mpim:read,mpim:write," +
	"pins:read,pins:write,reactions:read,reactions:write," +
	"search:read,stars:read,stars:write," +
	"team:read,users:read,users:read.email,users.profile:read,users.profile:write," +
	"reminders:read,reminders:write"

// Result holds the tokens and identity returned by the OAuth flow.
type Result struct {
	UserToken string
	AppToken  string // returned by proxy mode (Worker holds this secret)
	TeamID    string
	TeamName  string
	UserID    string
}

// Params configures the OAuth flow.
type Params struct {
	ClientID     string
	ClientSecret string // direct exchange (self-hosted mode)
	ProxyURL     string // Cloudflare Worker URL (public distribution mode)
	OpenBrowser  func(string) error
}

// flowResult is sent through the channel from callback handlers to Run().
type flowResult struct {
	result *Result
	err    error
}

// Run executes the local OAuth flow. There are two modes:
//
// Proxy mode (ProxyURL set): Browser → Worker/authorize → Slack → Worker/callback
// (exchanges token) → form POST to localhost/done.
//
// Direct mode (ClientSecret set): Browser → Slack → localhost/callback → direct
// token exchange with Slack API.
func Run(ctx context.Context, p Params) (*Result, error) {
	if p.ClientSecret == "" && p.ProxyURL == "" {
		return nil, fmt.Errorf("either client_secret (self-hosted) or proxy_url (public) must be configured")
	}

	openBrowser := p.OpenBrowser
	if openBrowser == nil {
		openBrowser = defaultOpenBrowser
	}

	// Listen on a random port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	ch := make(chan flowResult, 1)
	mux := http.NewServeMux()

	if p.ProxyURL != "" {
		mux.HandleFunc("/done", handleProxyDone(ch, state))
	} else {
		redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
		mux.HandleFunc("/callback", handleDirectCallback(ch, state, p.ClientID, p.ClientSecret, redirectURI))
	}

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(ln)
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Build the URL to open.
	var authURL string
	if p.ProxyURL != "" {
		authURL = buildProxyAuthorizeURL(p.ProxyURL, port, state)
	} else {
		redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
		authURL = buildAuthURL(p.ClientID, redirectURI, state)
	}

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Open this URL in your browser:\n%s\n", authURL)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	select {
	case res := <-ch:
		return res.result, res.err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("OAuth timed out waiting for authorization")
	}
}

// handleProxyDone returns an HTTP handler for POST /done, which receives
// the token and identity from the Worker's auto-submitted form.
func handleProxyDone(ch chan<- flowResult, expectedState string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			ch <- flowResult{err: fmt.Errorf("failed to parse form: %w", err)}
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if r.FormValue("state") != expectedState {
			ch <- flowResult{err: fmt.Errorf("state mismatch (possible CSRF)")}
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}

		token := r.FormValue("token")
		if token == "" {
			ch <- flowResult{err: fmt.Errorf("no token in callback")}
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}

		ch <- flowResult{result: &Result{
			UserToken: token,
			AppToken:  r.FormValue("app_token"),
			TeamID:    r.FormValue("team_id"),
			TeamName:  r.FormValue("team_name"),
			UserID:    r.FormValue("user_id"),
		}}

		fmt.Fprintf(w, "<html><body><h2>Authorization successful!</h2><p>You can close this window and return to Slacko.</p></body></html>")
	}
}

// handleDirectCallback returns an HTTP handler for GET /callback (direct mode),
// which receives the code from Slack and exchanges it locally.
func handleDirectCallback(ch chan<- flowResult, expectedState, clientID, clientSecret, redirectURI string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			ch <- flowResult{err: fmt.Errorf("slack denied authorization: %s", errMsg)}
			fmt.Fprintf(w, "<html><body><h2>Authorization denied.</h2><p>You can close this window.</p></body></html>")
			return
		}

		if r.URL.Query().Get("state") != expectedState {
			ch <- flowResult{err: fmt.Errorf("state mismatch (possible CSRF)")}
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			ch <- flowResult{err: fmt.Errorf("no code in callback")}
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}

		result, err := exchangeCodeDirect(clientID, clientSecret, code, redirectURI)
		ch <- flowResult{result: result, err: err}

		if err != nil {
			fmt.Fprintf(w, "<html><body><h2>Authorization failed.</h2><p>%s</p></body></html>", err.Error())
		} else {
			fmt.Fprintf(w, "<html><body><h2>Authorization successful!</h2><p>You can close this window and return to Slacko.</p></body></html>")
		}
	}
}

// generateState creates a random hex-encoded CSRF state token.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// buildAuthURL constructs the Slack OAuth v2 authorization URL (direct mode).
func buildAuthURL(clientID, redirectURI, state string) string {
	params := url.Values{
		"client_id":    {clientID},
		"user_scope":   {UserScopes},
		"redirect_uri": {redirectURI},
		"state":        {state},
	}
	return "https://slack.com/oauth/v2/authorize?" + params.Encode()
}

// buildProxyAuthorizeURL constructs the Worker /authorize URL (proxy mode).
func buildProxyAuthorizeURL(proxyURL string, port int, state string) string {
	base := strings.TrimRight(proxyURL, "/")
	params := url.Values{
		"port":  {fmt.Sprintf("%d", port)},
		"state": {state},
	}
	return base + "/authorize?" + params.Encode()
}

// exchangeCodeDirect exchanges the authorization code for an access token
// directly with the Slack API (self-hosted mode with client_secret).
func exchangeCodeDirect(clientID, clientSecret, code, redirectURI string) (*Result, error) {
	resp, err := slack.GetOAuthV2Response(http.DefaultClient, clientID, clientSecret, code, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	userToken := resp.AuthedUser.AccessToken
	if userToken == "" {
		return nil, fmt.Errorf("no user token in OAuth response")
	}

	return &Result{
		UserToken: userToken,
		TeamID:    resp.Team.ID,
		TeamName:  resp.Team.Name,
		UserID:    resp.AuthedUser.ID,
	}, nil
}

// defaultOpenBrowser opens a URL in the system's default browser.
func defaultOpenBrowser(rawURL string) error {
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open", rawURL}
	case "windows":
		args = []string{"rundll32", "url.dll,FileProtocolHandler", rawURL}
	default:
		args = []string{"xdg-open", rawURL}
	}

	if !strings.HasPrefix(rawURL, "https://") && !strings.HasPrefix(rawURL, "http://") {
		return fmt.Errorf("refusing to open non-HTTP URL")
	}

	return exec.Command(args[0], args[1:]...).Start()
}
