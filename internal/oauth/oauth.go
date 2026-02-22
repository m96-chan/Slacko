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
	TeamID    string
	TeamName  string
	UserID    string
}

// Run executes the local OAuth flow:
//  1. Listens on a random localhost port
//  2. Opens the browser to Slack's authorize endpoint
//  3. Receives the callback with the authorization code
//  4. Exchanges the code for an access token
//
// openBrowser is called with the authorization URL. If nil, the default
// system browser opener is used.
func Run(ctx context.Context, clientID, clientSecret, appToken string, openBrowser func(string) error) (*Result, error) {
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
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	authURL := buildAuthURL(clientID, redirectURI, state)

	// Channel to receive the result from the callback handler.
	type callbackResult struct {
		code string
		err  error
	}
	ch := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			ch <- callbackResult{err: fmt.Errorf("slack denied authorization: %s", errMsg)}
			fmt.Fprintf(w, "<html><body><h2>Authorization denied.</h2><p>You can close this window.</p></body></html>")
			return
		}

		if r.URL.Query().Get("state") != state {
			ch <- callbackResult{err: fmt.Errorf("state mismatch (possible CSRF)")}
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			ch <- callbackResult{err: fmt.Errorf("no code in callback")}
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}

		ch <- callbackResult{code: code}
		fmt.Fprintf(w, "<html><body><h2>Authorization successful!</h2><p>You can close this window and return to Slacko.</p></body></html>")
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(ln)
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Open browser (best-effort; print URL on failure).
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Open this URL in your browser:\n%s\n", authURL)
	}

	// Wait for callback or timeout.
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		return exchangeCode(clientID, clientSecret, res.code, redirectURI)
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("OAuth timed out waiting for authorization")
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

// buildAuthURL constructs the Slack OAuth v2 authorization URL.
func buildAuthURL(clientID, redirectURI, state string) string {
	params := url.Values{
		"client_id":    {clientID},
		"user_scope":   {UserScopes},
		"redirect_uri": {redirectURI},
		"state":        {state},
	}
	return "https://slack.com/oauth/v2/authorize?" + params.Encode()
}

// exchangeCode exchanges the authorization code for an access token using
// the slack-go library.
func exchangeCode(clientID, clientSecret, code, redirectURI string) (*Result, error) {
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
func defaultOpenBrowser(url string) error {
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open", url}
	case "windows":
		args = []string{"rundll32", "url.dll,FileProtocolHandler", url}
	default:
		args = []string{"xdg-open", url}
	}

	// Validate the URL scheme to prevent command injection.
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return fmt.Errorf("refusing to open non-HTTP URL")
	}

	return exec.Command(args[0], args[1:]...).Start()
}
