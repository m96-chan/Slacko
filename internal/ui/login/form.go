package login

import (
	"log/slog"

	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/keyring"
	slackclient "github.com/m96-chan/Slacko/internal/slack"
)

// DoneFn is called after successful authentication with the validated client.
type DoneFn func(client *slackclient.Client)

// Form is a tview form that prompts for Slack tokens.
type Form struct {
	*tview.Form
	app      *tview.Application
	cfg      *config.Config
	done     DoneFn
	botField *tview.InputField
	appField *tview.InputField
}

// New creates a login form with bot-token and app-token password fields.
func New(app *tview.Application, cfg *config.Config, done DoneFn) *Form {
	f := &Form{
		Form: tview.NewForm(),
		app:  app,
		cfg:  cfg,
		done: done,
	}

	f.botField = tview.NewInputField().
		SetLabel("Bot Token (xoxb-)").
		SetMaskCharacter('*')
	f.appField = tview.NewInputField().
		SetLabel("App Token (xapp-)").
		SetMaskCharacter('*')

	f.AddFormItem(f.botField).
		AddFormItem(f.appField).
		AddButton("Login", f.submit).
		AddButton("Quit", func() { app.Stop() }).
		SetBorder(true).
		SetTitle(" slacko login ").
		SetTitleAlign(tview.AlignCenter)

	return f
}

// submit validates the tokens, creates a client, saves to keyring, and
// calls the done callback.
func (f *Form) submit() {
	bot := f.botField.GetText()
	app := f.appField.GetText()

	if bot == "" || app == "" {
		f.showError("Both tokens are required.")
		return
	}

	client, err := slackclient.New(bot, app)
	if err != nil {
		f.showError("Authentication failed: " + err.Error())
		return
	}

	if err := keyring.SetBotToken(bot); err != nil {
		slog.Warn("failed to store bot token in keyring", "error", err)
	}
	if err := keyring.SetAppToken(app); err != nil {
		slog.Warn("failed to store app token in keyring", "error", err)
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
