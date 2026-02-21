package config

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// StyleWrapper wraps tcell.Style and implements TOML unmarshalling.
// In TOML it is represented as a table with optional "foreground",
// "background", and "attributes" string fields.
type StyleWrapper struct {
	tcell.Style
}

// styleTable is the intermediate representation used for TOML unmarshalling.
type styleTable struct {
	Foreground string `toml:"foreground"`
	Background string `toml:"background"`
	Attributes string `toml:"attributes"`
}

// UnmarshalTOML implements the toml.Unmarshaler interface.
func (s *StyleWrapper) UnmarshalTOML(data any) error {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("expected table for style, got %T", data)
	}

	style := tcell.StyleDefault

	if fg, ok := m["foreground"].(string); ok && fg != "" {
		style = style.Foreground(tcell.GetColor(fg))
	}
	if bg, ok := m["background"].(string); ok && bg != "" {
		style = style.Background(tcell.GetColor(bg))
	}
	if attrs, ok := m["attributes"].(string); ok && attrs != "" {
		mask, err := stringToAttrMask(attrs)
		if err != nil {
			return err
		}
		style = style.Attributes(mask)
	}

	s.Style = style
	return nil
}

// stringToAttrMask parses a pipe-separated list of attribute names into
// a tcell.AttrMask. For example: "bold|underline".
func stringToAttrMask(s string) (tcell.AttrMask, error) {
	var mask tcell.AttrMask
	for _, part := range strings.Split(s, "|") {
		part = strings.TrimSpace(strings.ToLower(part))
		switch part {
		case "bold":
			mask |= tcell.AttrBold
		case "italic":
			mask |= tcell.AttrItalic
		case "underline":
			mask |= tcell.AttrUnderline
		case "dim":
			mask |= tcell.AttrDim
		case "reverse":
			mask |= tcell.AttrReverse
		case "blink":
			mask |= tcell.AttrBlink
		case "strikethrough":
			mask |= tcell.AttrStrikeThrough
		case "none", "":
			// no-op
		default:
			return 0, fmt.Errorf("unknown style attribute: %q", part)
		}
	}
	return mask, nil
}

// Theme holds the complete theme configuration.
type Theme struct {
	Border       BorderTheme       `toml:"border"`
	Title        TitleTheme        `toml:"title"`
	ChannelsTree ChannelsTreeTheme `toml:"channels_tree"`
	MessagesList MessagesListTheme `toml:"messages_list"`
	MessageInput MessageInputTheme `toml:"message_input"`
	StatusBar    StatusBarTheme    `toml:"status_bar"`
}

// BorderTheme configures border styling.
type BorderTheme struct {
	Focused StyleWrapper `toml:"focused"`
	Normal  StyleWrapper `toml:"normal"`
}

// TitleTheme configures title bar styling.
type TitleTheme struct {
	Focused StyleWrapper `toml:"focused"`
	Normal  StyleWrapper `toml:"normal"`
}

// ChannelsTreeTheme configures the channels tree styling.
type ChannelsTreeTheme struct {
	Channel  StyleWrapper `toml:"channel"`
	Selected StyleWrapper `toml:"selected"`
	Unread   StyleWrapper `toml:"unread"`
}

// MessagesListTheme configures the messages list styling.
type MessagesListTheme struct {
	Message   StyleWrapper `toml:"message"`
	Author    StyleWrapper `toml:"author"`
	Timestamp StyleWrapper `toml:"timestamp"`
	Selected  StyleWrapper `toml:"selected"`
	Reply     StyleWrapper `toml:"reply"`
}

// MessageInputTheme configures the message input styling.
type MessageInputTheme struct {
	Text        StyleWrapper `toml:"text"`
	Placeholder StyleWrapper `toml:"placeholder"`
}

// StatusBarTheme configures the status bar styling.
type StatusBarTheme struct {
	Text       StyleWrapper `toml:"text"`
	Background StyleWrapper `toml:"background"`
}
