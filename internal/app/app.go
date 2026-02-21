package app

import (
	"context"
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
	cancel context.CancelFunc
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

// showMain sets the root to the main view and starts Socket Mode in the
// background. Currently a placeholder that displays the authenticated identity
// and connection status.
func (a *App) showMain() {
	// Cancel any previous Socket Mode connection.
	if a.cancel != nil {
		a.cancel()
	}

	text := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("authenticated as %s (%s) — connecting...", a.slack.UserName, a.slack.TeamName))

	a.tview.SetRoot(text, true)

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	handler := &slackclient.EventHandler{
		OnConnected: func() {
			slog.Info("socket mode connected")
			a.tview.QueueUpdateDraw(func() {
				text.SetText(fmt.Sprintf("authenticated as %s (%s) — connected", a.slack.UserName, a.slack.TeamName))
			})
		},
		OnDisconnected: func() {
			slog.Warn("socket mode disconnected")
			a.tview.QueueUpdateDraw(func() {
				text.SetText(fmt.Sprintf("authenticated as %s (%s) — disconnected", a.slack.UserName, a.slack.TeamName))
			})
		},
		OnError: func(err error) {
			slog.Error("socket mode error", "error", err)
		},
	}

	go func() {
		if err := a.slack.RunSocketMode(ctx, handler); err != nil {
			slog.Error("socket mode exited", "error", err)
		}
	}()
}
