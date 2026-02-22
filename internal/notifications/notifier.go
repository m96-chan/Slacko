package notifications

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// minInterval is the minimum time between notifications to prevent spam.
const minInterval = 3 * time.Second

// Notifier sends desktop notifications with rate limiting.
type Notifier struct {
	mu       sync.Mutex
	lastSent time.Time
}

// New creates a new Notifier.
func New() *Notifier {
	return &Notifier{}
}

// Send dispatches a desktop notification using the platform's native system.
// Returns silently if rate-limited or if the platform is unsupported.
func (n *Notifier) Send(title, body string) {
	n.mu.Lock()
	if time.Since(n.lastSent) < minInterval {
		n.mu.Unlock()
		return
	}
	n.lastSent = time.Now()
	n.mu.Unlock()

	go func() {
		if err := sendPlatform(title, body); err != nil {
			slog.Debug("notification failed", "error", err)
		}
	}()
}

// sendPlatform dispatches a notification using OS-specific commands.
func sendPlatform(title, body string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("notify-send", "--app-name=Slacko", title, body).Run()
	case "darwin":
		script := fmt.Sprintf(
			`display notification %q with title %q`, body, title)
		return exec.Command("osascript", "-e", script).Run()
	default:
		slog.Debug("notifications not supported on this platform", "os", runtime.GOOS)
		return nil
	}
}

// MentionType identifies what kind of notification trigger was detected.
type MentionType int

const (
	MentionNone MentionType = iota
	MentionDirect
	MentionDM
	MentionHere
	MentionChannel
	MentionEveryone
)

// DetectMention checks if a message should trigger a notification for the given user.
// Returns the most specific mention type found.
func DetectMention(text, selfUserID string, isDM bool) MentionType {
	if isDM {
		return MentionDM
	}

	// Direct @mention: <@USERID> or <@USERID|name>.
	selfMention := "<@" + selfUserID + ">"
	selfMentionPipe := "<@" + selfUserID + "|"
	if strings.Contains(text, selfMention) || strings.Contains(text, selfMentionPipe) {
		return MentionDirect
	}

	// Group mentions.
	if strings.Contains(text, "<!everyone>") {
		return MentionEveryone
	}
	if strings.Contains(text, "<!channel>") {
		return MentionChannel
	}
	if strings.Contains(text, "<!here>") {
		return MentionHere
	}

	return MentionNone
}

// StripMrkdwn removes Slack mrkdwn formatting for plain-text notification body.
// Resolves mentions to readable text and strips formatting tokens.
func StripMrkdwn(text string) string {
	// Remove bold/italic/strikethrough markers.
	for _, ch := range []string{"*", "_", "~"} {
		text = strings.ReplaceAll(text, ch, "")
	}

	// Remove code block fences.
	text = strings.ReplaceAll(text, "```", "")

	// Remove inline code backticks.
	text = strings.ReplaceAll(text, "`", "")

	// Resolve <@U123|name> → @name, <@U123> → @U123.
	text = resolveAngleBrackets(text)

	// Truncate long messages.
	if len(text) > 200 {
		text = text[:200] + "…"
	}

	return strings.TrimSpace(text)
}

// resolveAngleBrackets converts <...> tokens to readable text.
func resolveAngleBrackets(text string) string {
	var result strings.Builder
	i := 0
	for i < len(text) {
		if text[i] == '<' {
			end := strings.IndexByte(text[i:], '>')
			if end < 0 {
				result.WriteByte(text[i])
				i++
				continue
			}
			inner := text[i+1 : i+end]
			result.WriteString(resolveToken(inner))
			i += end + 1
		} else {
			result.WriteByte(text[i])
			i++
		}
	}
	return result.String()
}

// resolveToken converts a single angle-bracket token to readable text.
func resolveToken(inner string) string {
	switch {
	case strings.HasPrefix(inner, "@"):
		parts := strings.SplitN(inner[1:], "|", 2)
		if len(parts) == 2 && parts[1] != "" {
			return "@" + parts[1]
		}
		return "@" + parts[0]
	case strings.HasPrefix(inner, "#"):
		parts := strings.SplitN(inner[1:], "|", 2)
		if len(parts) == 2 && parts[1] != "" {
			return "#" + parts[1]
		}
		return "#" + parts[0]
	case strings.HasPrefix(inner, "!"):
		parts := strings.SplitN(inner[1:], "|", 2)
		if len(parts) == 2 && parts[1] != "" {
			return parts[1]
		}
		return "@" + parts[0]
	default:
		// URL or URL|label.
		parts := strings.SplitN(inner, "|", 2)
		if len(parts) == 2 && parts[1] != "" {
			return parts[1]
		}
		return parts[0]
	}
}
