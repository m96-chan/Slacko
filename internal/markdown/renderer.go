package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/rivo/tview"
	"github.com/slack-go/slack"
)

// placeholder markers for tokens that should not be processed by inline formatting.
const placeholderPrefix = "\x00T"
const placeholderSuffix = "\x00"

// Compiled patterns for Slack mrkdwn.
var (
	// Slack angle-bracket tokens: <@U123>, <@U123|name>, <#C123>, <#C123|name>,
	// <!here>, <!channel>, <!everyone>, <URL>, <URL|label>.
	slackTokenRe = regexp.MustCompile(`<([^>]+)>`)

	// Inline code: `text` (single backtick, not inside code blocks).
	inlineCodeRe = regexp.MustCompile("`([^`\n]+)`")

	// Bold: *text* (content must not start/end with space).
	boldRe = regexp.MustCompile(`\*([^\*\n]+)\*`)

	// Italic: _text_ (content must not start/end with space).
	italicRe = regexp.MustCompile(`_([^_\n]+)_`)

	// Strikethrough: ~text~.
	strikeRe = regexp.MustCompile(`~([^~\n]+)~`)
	// Emoji: :name: (alphanumeric, underscore, hyphen, plus).
	emojiRe = regexp.MustCompile(`:([a-zA-Z0-9_+\-]+):`)

	// Code block: ```lang\ncode``` or ```code```.
	codeBlockRe = regexp.MustCompile("(?s)```(\\w*)\\n?(.*?)```")
)

// MarkdownColors holds pre-computed tview tag strings for markdown rendering,
// avoiding a direct dependency on the config package.
type MarkdownColors struct {
	UserMention    string // e.g. "[yellow::b]"
	ChannelMention string // e.g. "[cyan::b]"
	SpecialMention string // e.g. "[yellow::bu]"
	Link           string // e.g. "[blue::u]"
	InlineCode     string // e.g. "[gray]"
	CodeFence      string // e.g. "[gray]"
	BlockquoteMark string // e.g. "[gray]"
	BlockquoteText string // e.g. "[::d]"
}

// DefaultMarkdownColors returns colors matching the original hardcoded values.
func DefaultMarkdownColors() MarkdownColors {
	return MarkdownColors{
		UserMention:    "[yellow::b]",
		ChannelMention: "[cyan::b]",
		SpecialMention: "[yellow::bu]",
		Link:           "[blue::u]",
		InlineCode:     "[gray]",
		CodeFence:      "[gray]",
		BlockquoteMark: "[gray]",
		BlockquoteText: "[::d]",
	}
}

// Render converts Slack mrkdwn text to tview-formatted output.
// When enabled is false, only basic token resolution is performed (user/channel
// mentions) with all output escaped for tview.
func Render(text string, users map[string]slack.User, channels map[string]string, enabled bool, syntaxTheme string, colors MarkdownColors) string {
	if !enabled {
		text = resolveSlackTokens(text, users, channels, false)
		return escapeLines(text)
	}

	// Split text into code blocks and non-code segments.
	segments := splitCodeBlocks(text)

	var b strings.Builder
	for _, seg := range segments {
		if seg.isCode {
			b.WriteString(renderCodeBlock(seg.lang, seg.code, syntaxTheme, colors))
		} else {
			b.WriteString(renderInline(seg.text, users, channels, colors))
		}
	}

	return b.String()
}

// segment represents either a code block or inline text.
type segment struct {
	isCode bool
	lang   string // language hint for code blocks
	code   string // code block content
	text   string // inline text content
}

// splitCodeBlocks splits text into alternating inline/code-block segments.
func splitCodeBlocks(text string) []segment {
	var segments []segment

	matches := codeBlockRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return []segment{{text: text}}
	}

	prev := 0
	for _, m := range matches {
		// Text before this code block.
		if m[0] > prev {
			segments = append(segments, segment{text: text[prev:m[0]]})
		}

		lang := text[m[2]:m[3]]
		code := text[m[4]:m[5]]

		segments = append(segments, segment{
			isCode: true,
			lang:   lang,
			code:   code,
		})
		prev = m[1]
	}

	// Remaining text after last code block.
	if prev < len(text) {
		segments = append(segments, segment{text: text[prev:]})
	}

	return segments
}

// renderCodeBlock renders a fenced code block with syntax highlighting.
func renderCodeBlock(lang, code string, syntaxTheme string, colors MarkdownColors) string {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get(syntaxTheme)
	if style == nil {
		style = styles.Fallback
	}

	fenceTag := colors.CodeFence
	fenceReset := "[-]"
	if strings.Count(fenceTag, ":") >= 2 {
		fenceReset = "[-::-]"
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		// Fallback: just escape and show as-is.
		return fenceTag + "```" + fenceReset + "\n" + tview.Escape(code) + "\n" + fenceTag + "```" + fenceReset
	}

	var buf strings.Builder
	buf.WriteString(fenceTag + "```" + fenceReset)
	if lang != "" {
		buf.WriteString(fenceTag + tview.Escape(lang) + fenceReset)
	}
	buf.WriteString("\n")

	for _, token := range iterator.Tokens() {
		text := tview.Escape(token.Value)
		entry := style.Get(token.Type)

		if entry.Colour.IsSet() {
			hex := entry.Colour.String()
			attrs := ""
			if entry.Bold == chroma.Yes {
				attrs = "b"
			}
			if entry.Italic == chroma.Yes {
				if attrs != "" {
					attrs += "i"
				} else {
					attrs = "i"
				}
			}

			if attrs != "" {
				fmt.Fprintf(&buf, "[%s::%s]%s[-::-]", hex, attrs, text)
			} else {
				fmt.Fprintf(&buf, "[%s]%s[-]", hex, text)
			}
		} else {
			buf.WriteString(text)
		}
	}

	// Remove trailing newline from chroma output before adding closing marker.
	result := buf.String()
	result = strings.TrimRight(result, "\n")
	return result + "\n" + fenceTag + "```" + fenceReset
}

// renderInline processes inline mrkdwn formatting.
func renderInline(text string, users map[string]slack.User, channels map[string]string, colors MarkdownColors) string {
	// Step 1: Extract <...> tokens and replace with placeholders.
	var placeholders []string
	text = slackTokenRe.ReplaceAllStringFunc(text, func(match string) string {
		inner := match[1 : len(match)-1]
		rendered := renderSlackToken(inner, users, channels, colors)
		idx := len(placeholders)
		placeholders = append(placeholders, rendered)
		return fmt.Sprintf("%s%d%s", placeholderPrefix, idx, placeholderSuffix)
	})

	// Step 2: Escape tview special chars in remaining text.
	text = tview.Escape(text)

	// Step 3: Extract inline code with placeholders (prevents formatting inside code).
	inlineCodeTag := colors.InlineCode
	inlineCodeReset := "[-]"
	if strings.Count(inlineCodeTag, ":") >= 2 {
		inlineCodeReset = "[-::-]"
	}
	text = inlineCodeRe.ReplaceAllStringFunc(text, func(match string) string {
		content := match[1 : len(match)-1] // strip backticks
		rendered := inlineCodeTag + "`" + content + "`" + inlineCodeReset
		idx := len(placeholders)
		placeholders = append(placeholders, rendered)
		return fmt.Sprintf("%s%d%s", placeholderPrefix, idx, placeholderSuffix)
	})

	// Step 4: Process blockquotes (line-level, before inline formatting).
	text = renderBlockquotes(text, colors)

	// Step 5: Process formatting: *bold*, _italic_, ~strikethrough~.
	text = boldRe.ReplaceAllString(text, "[::b]$1[::-]")
	text = italicRe.ReplaceAllString(text, "[::i]$1[::-]")
	text = strikeRe.ReplaceAllString(text, "[::s]$1[::-]")

	// Step 6: Process emoji :name:.
	text = emojiRe.ReplaceAllStringFunc(text, func(match string) string {
		name := match[1 : len(match)-1]
		return lookupEmoji(name)
	})

	// Step 7: Restore placeholders.
	for i, p := range placeholders {
		placeholder := fmt.Sprintf("%s%d%s", placeholderPrefix, i, placeholderSuffix)
		text = strings.Replace(text, placeholder, p, 1)
	}

	return text
}

// renderBlockquotes converts lines starting with "> " to styled blockquotes.
func renderBlockquotes(text string, colors MarkdownColors) string {
	markTag := colors.BlockquoteMark
	markReset := "[-]"
	if strings.Count(markTag, ":") >= 2 {
		markReset = "[-::-]"
	}
	textTag := colors.BlockquoteText
	textReset := "[::-]"

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := line
		// Strip leading whitespace for quote detection.
		stripped := strings.TrimLeft(trimmed, " \t")
		if strings.HasPrefix(stripped, "&gt; ") {
			// "&gt;" is the HTML entity for ">" which tview.Escape may produce — but
			// actually tview.Escape only escapes [...] sequences. We check for literal ">".
			continue
		}
		if strings.HasPrefix(stripped, "> ") {
			content := stripped[2:]
			lines[i] = markTag + "▎" + markReset + " " + textTag + content + textReset
		} else if stripped == ">" {
			lines[i] = markTag + "▎" + markReset
		}
	}
	return strings.Join(lines, "\n")
}

// renderSlackToken converts a single Slack angle-bracket token to styled text.
func renderSlackToken(inner string, users map[string]slack.User, channels map[string]string, colors MarkdownColors) string {
	switch {
	// User mention: @U123 or @U123|name.
	case strings.HasPrefix(inner, "@"):
		return renderUserMention(inner[1:], users, colors)

	// Channel mention: #C123 or #C123|name.
	case strings.HasPrefix(inner, "#"):
		return renderChannelMention(inner[1:], channels, colors)

	// Special mentions: !here, !channel, !everyone.
	case strings.HasPrefix(inner, "!"):
		return renderSpecialMention(inner[1:], colors)

	// URL: http(s)://... or URL|label.
	default:
		return renderLink(inner, colors)
	}
}

// renderUserMention renders @U123 or @U123|label.
func renderUserMention(token string, users map[string]slack.User, colors MarkdownColors) string {
	parts := strings.SplitN(token, "|", 2)
	userID := parts[0]

	var name string
	if len(parts) == 2 && parts[1] != "" {
		name = parts[1]
	} else if u, ok := users[userID]; ok {
		name = u.Profile.DisplayName
		if name == "" {
			name = u.Name
		}
		if name == "" {
			name = userID
		}
	} else {
		name = userID
	}

	return colors.UserMention + "@" + tview.Escape(name) + "[-::-]"
}

// renderChannelMention renders #C123 or #C123|name.
func renderChannelMention(token string, channels map[string]string, colors MarkdownColors) string {
	parts := strings.SplitN(token, "|", 2)
	channelID := parts[0]

	var name string
	if len(parts) == 2 && parts[1] != "" {
		name = parts[1]
	} else if n, ok := channels[channelID]; ok {
		name = n
	} else {
		name = channelID
	}

	return colors.ChannelMention + "#" + tview.Escape(name) + "[-::-]"
}

// renderSpecialMention renders !here, !channel, !everyone.
func renderSpecialMention(token string, colors MarkdownColors) string {
	parts := strings.SplitN(token, "|", 2)
	keyword := parts[0]

	var label string
	if len(parts) == 2 && parts[1] != "" {
		label = parts[1]
	} else {
		label = "@" + keyword
	}

	return colors.SpecialMention + tview.Escape(label) + "[-::-]"
}

// renderLink renders a URL or URL|label.
func renderLink(token string, colors MarkdownColors) string {
	parts := strings.SplitN(token, "|", 2)
	url := parts[0]

	if len(parts) == 2 && parts[1] != "" {
		label := parts[1]
		return colors.Link + tview.Escape(label) + "[-::-]"
	}

	return colors.Link + tview.Escape(url) + "[-::-]"
}

// resolveSlackTokens resolves <...> tokens without applying any formatting.
// Used when markdown rendering is disabled.
func resolveSlackTokens(text string, users map[string]slack.User, channels map[string]string, styled bool) string {
	return slackTokenRe.ReplaceAllStringFunc(text, func(match string) string {
		inner := match[1 : len(match)-1]

		switch {
		case strings.HasPrefix(inner, "@"):
			return resolveUserMentionPlain(inner[1:], users)
		case strings.HasPrefix(inner, "#"):
			return resolveChannelMentionPlain(inner[1:], channels)
		case strings.HasPrefix(inner, "!"):
			parts := strings.SplitN(inner[1:], "|", 2)
			if len(parts) == 2 && parts[1] != "" {
				return parts[1]
			}
			return "@" + parts[0]
		default:
			// URL|label or plain URL.
			parts := strings.SplitN(inner, "|", 2)
			if len(parts) == 2 && parts[1] != "" {
				return parts[1]
			}
			return parts[0]
		}
	})
}

// resolveUserMentionPlain returns @displayname for a user mention token.
func resolveUserMentionPlain(token string, users map[string]slack.User) string {
	parts := strings.SplitN(token, "|", 2)
	userID := parts[0]

	if len(parts) == 2 && parts[1] != "" {
		return "@" + parts[1]
	}
	if u, ok := users[userID]; ok {
		name := u.Profile.DisplayName
		if name == "" {
			name = u.Name
		}
		if name == "" {
			name = userID
		}
		return "@" + name
	}
	return "@" + userID
}

// resolveChannelMentionPlain returns #name for a channel mention token.
func resolveChannelMentionPlain(token string, channels map[string]string) string {
	parts := strings.SplitN(token, "|", 2)
	channelID := parts[0]

	if len(parts) == 2 && parts[1] != "" {
		return "#" + parts[1]
	}
	if name, ok := channels[channelID]; ok {
		return "#" + name
	}
	return "#" + channelID
}

// escapeLines escapes each line for tview display.
func escapeLines(text string) string {
	return tview.Escape(text)
}
