package app

import (
	"errors"
	"fmt"
	"log/slog"

	gokeyring "github.com/zalando/go-keyring"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/keyring"
	slackclient "github.com/m96-chan/Slacko/internal/slack"
	"github.com/m96-chan/Slacko/internal/ui/login"
	"github.com/rivo/tview"
)

// App is the top-level application struct.
type App struct {
	Config *config.Config
	tview  *tview.Application
	slack  *slackclient.Client
}

// New creates a new App with the given config.
func New(cfg *config.Config) *App {
	return &App{
		Config: cfg,
		tview:  tview.NewApplication(),
	}
}

// Run starts the TUI event loop. It attempts to authenticate using stored
// tokens and shows the login form when tokens are missing or invalid.
func (a *App) Run() error {
	a.tview.EnableMouse(a.Config.Mouse)

	bot, botErr := keyring.GetBotToken()
	app, appErr := keyring.GetAppToken()

	if botErr == nil && appErr == nil {
		client, err := slackclient.New(bot, app)
		if err != nil {
			slog.Warn("stored tokens invalid, showing login", "error", err)
			a.showLogin()
		} else {
			a.slack = client
			a.showMain()
		}
	} else {
		if botErr != nil && !errors.Is(botErr, gokeyring.ErrNotFound) {
			slog.Warn("error reading bot token", "error", botErr)
		}
		if appErr != nil && !errors.Is(appErr, gokeyring.ErrNotFound) {
			slog.Warn("error reading app token", "error", appErr)
		}
		a.showLogin()
	}

	return a.tview.Run()
}

// showLogin sets the root to the login form.
func (a *App) showLogin() {
	form := login.New(a.tview, a.Config, func(client *slackclient.Client) {
		a.slack = client
		a.showMain()
	})
	a.tview.SetRoot(form, true)
}

// showMain sets the root to the main view. Currently a placeholder that
// displays the authenticated identity.
func (a *App) showMain() {
	text := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("authenticated as %s (%s)", a.slack.UserName, a.slack.TeamName))

	a.tview.SetRoot(text, true)
}
