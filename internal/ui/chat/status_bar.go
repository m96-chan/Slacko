package chat

import (
	"fmt"

	"github.com/m96-chan/Slacko/internal/config"
	"github.com/rivo/tview"
)

// StatusBar displays connection status and typing indicator at the bottom.
type StatusBar struct {
	*tview.TextView
	cfg          *config.Config
	connStatus   string
	typingText   string
	presenceText string
}

// NewStatusBar creates a themed status bar.
func NewStatusBar(cfg *config.Config) *StatusBar {
	tv := tview.NewTextView().
		SetDynamicColors(true)

	// Apply status bar theme.
	_, bg, _ := cfg.Theme.StatusBar.Background.Style.Decompose()
	fg, _, _ := cfg.Theme.StatusBar.Text.Style.Decompose()
	tv.SetBackgroundColor(bg)
	tv.SetTextColor(fg)

	sb := &StatusBar{
		TextView: tv,
		cfg:      cfg,
	}
	return sb
}

// SetConnectionStatus updates the connection status text.
func (sb *StatusBar) SetConnectionStatus(s string) {
	sb.connStatus = s
	sb.render()
}

// SetTypingIndicator updates the typing indicator text.
func (sb *StatusBar) SetTypingIndicator(s string) {
	sb.typingText = s
	sb.render()
}

// SetChannelPresence updates the online member count display.
func (sb *StatusBar) SetChannelPresence(online, total int) {
	if total > 0 {
		sb.presenceText = fmt.Sprintf("%d/%d online", online, total)
	} else {
		sb.presenceText = ""
	}
	sb.render()
}

// render rebuilds the status bar text from current state.
func (sb *StatusBar) render() {
	text := " " + sb.connStatus
	if sb.presenceText != "" {
		text += "  |  " + sb.presenceText
	}
	if sb.typingText != "" {
		text += "  |  " + sb.typingText
	}
	sb.TextView.SetText(text)
}
