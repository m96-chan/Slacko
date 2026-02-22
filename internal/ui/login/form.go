package login

import (
	"context"
	"log/slog"

	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/keyring"
	"github.com/m96-chan/Slacko/internal/oauth"
	slackclient "github.com/m96-chan/Slacko/internal/slack"
)

// DoneFn is called after successful authentication with the validated client.
type DoneFn func(client *slackclient.Client)

// Form is a tview form that prompts for Slack tokens.
type Form struct {
	*tview.Form
	app       *tview.Application
	cfg       *config.Config
	done      DoneFn
	userField *tview.InputField
	appField  *tview.InputField
}

// New creates a login form. If OAuth credentials (client_id) are configured,
// it shows a browser-based OAuth login. Otherwise it falls back to the manual
// token input form.
func New(app *tview.Application, cfg *config.Config, done DoneFn) *Form {
	f := &Form{
		Form: tview.NewForm(),
		app:  app,
		cfg:  cfg,
		done: done,
	}

	if cfg.OAuth.ClientID != "" {
		f.buildOAuthForm()
	} else {
		f.buildManualForm()
	}

	return f
}

// buildOAuthForm sets up a single-button form that triggers the browser OAuth flow.
func (f *Form) buildOAuthForm() {
	f.AddButton("Authorize with Slack (opens browser)", f.submitOAuth).
		AddButton("Quit", func() { f.app.Stop() }).
		SetBorder(true).
		SetTitle(" slacko login — OAuth ").
		SetTitleAlign(tview.AlignCenter)
}

// buildManualForm sets up the traditional token-paste form.
func (f *Form) buildManualForm() {
	f.userField = tview.NewInputField().
		SetLabel("User Token (xoxp-)").
		SetMaskCharacter('*')
	f.appField = tview.NewInputField().
		SetLabel("App Token (xapp-)").
		SetMaskCharacter('*')

	f.AddFormItem(f.userField).
		AddFormItem(f.appField).
		AddButton("Login", f.submitManual).
		AddButton("Quit", func() { f.app.Stop() }).
		SetBorder(true).
		SetTitle(" slacko login ").
		SetTitleAlign(tview.AlignCenter)
}

// submitOAuth suspends the TUI, runs the OAuth flow, and resumes.
func (f *Form) submitOAuth() {
	cfg := f.cfg.OAuth

	var result *oauth.Result
	var oauthErr error

	f.app.Suspend(func() {
		result, oauthErr = oauth.Run(
			context.Background(),
			cfg.ClientID,
			cfg.ClientSecret,
			cfg.AppToken,
			nil, // use default browser opener
		)
	})

	if oauthErr != nil {
		f.showError("OAuth failed: " + oauthErr.Error())
		return
	}

	appToken := cfg.AppToken
	client, err := slackclient.New(result.UserToken, appToken)
	if err != nil {
		f.showError("Authentication failed: " + err.Error())
		return
	}

	if err := keyring.SetUserToken(result.UserToken); err != nil {
		slog.Warn("failed to store user token in keyring", "error", err)
	}
	if err := keyring.SetAppToken(appToken); err != nil {
		slog.Warn("failed to store app token in keyring", "error", err)
	}

	if err := keyring.AddWorkspace(client.TeamID, client.TeamName, result.UserToken, appToken); err != nil {
		slog.Warn("failed to register workspace", "error", err)
	}

	f.done(client)
}

// submitManual validates the tokens, creates a client, saves to keyring, and
// calls the done callback.
func (f *Form) submitManual() {
	user := f.userField.GetText()
	app := f.appField.GetText()

	if user == "" || app == "" {
		f.showError("Both tokens are required.")
		return
	}

	client, err := slackclient.New(user, app)
	if err != nil {
		f.showError("Authentication failed: " + err.Error())
		return
	}

	if err := keyring.SetUserToken(user); err != nil {
		slog.Warn("failed to store user token in keyring", "error", err)
	}
	if err := keyring.SetAppToken(app); err != nil {
		slog.Warn("failed to store app token in keyring", "error", err)
	}

	// Register in multi-workspace registry.
	if err := keyring.AddWorkspace(client.TeamID, client.TeamName, user, app); err != nil {
		slog.Warn("failed to register workspace", "error", err)
	}

	f.done(client)
}

// showError displays a modal error message and returns to the form on dismiss.
func (f *Form) showError(msg string) {
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			f.app.SetRoot(f, true)
		})
	f.app.SetRoot(modal, true)
}
