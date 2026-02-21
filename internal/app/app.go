package app

import (
	"github.com/m96-chan/Slacko/internal/config"
	"github.com/rivo/tview"
)

// App is the top-level application struct.
type App struct {
	Config *config.Config
	tview  *tview.Application
}

// New creates a new App with the given config.
func New(cfg *config.Config) *App {
	return &App{
		Config: cfg,
		tview:  tview.NewApplication(),
	}
}

// Run starts the TUI event loop. It shows a placeholder view until
// the Slack client is wired in a future issue.
func (a *App) Run() error {
	text := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("slacko - scaffolding complete")

	a.tview.SetRoot(text, true)
	a.tview.EnableMouse(a.Config.Mouse)

	return a.tview.Run()
}
