package markdown

import (
	"strings"
	"sync"
	"unicode"

	"github.com/kyokomi/emoji/v2"
)

var (
	emojiEntries     map[string]string
	emojiEntriesOnce sync.Once
)

// buildEmojiEntries creates the name→unicode map from kyokomi/emoji,
// filtering to Slack-compatible shortcodes (lowercase, digits, _, -, +).
func buildEmojiEntries() map[string]string {
	codeMap := emoji.CodeMap()
	result := make(map[string]string, len(codeMap))
	for k, v := range codeMap {
		name := strings.TrimPrefix(strings.TrimSuffix(k, ":"), ":")
		if !isSlackShortcode(name) {
			continue
		}
		result[name] = v
	}
	return result
}

// isSlackShortcode returns true if the name contains only lowercase letters,
// digits, underscores, hyphens, and plus signs (matching Slack shortcode conventions).
func isSlackShortcode(name string) bool {
	for _, r := range name {
		if !unicode.IsLower(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '+' {
			return false
		}
	}
	return len(name) > 0
}

func getEmojiEntries() map[string]string {
	emojiEntriesOnce.Do(func() {
		emojiEntries = buildEmojiEntries()
	})
	return emojiEntries
}

// lookupEmoji returns the unicode emoji for a name, or the :name: fallback.
// The name parameter should be without surrounding colons.
func lookupEmoji(name string) string {
	if u, ok := getEmojiEntries()[name]; ok {
		return u
	}
	return ":" + name + ":"
}

// LookupEmoji returns the unicode emoji for a name, or the :name: fallback.
func LookupEmoji(name string) string {
	return lookupEmoji(name)
}

// EmojiEntries returns all emoji name→unicode pairs.
// Keys are without surrounding colons.
func EmojiEntries() map[string]string {
	return getEmojiEntries()
}
