package config

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// StyleWrapper wraps tcell.Style and implements TOML unmarshalling.
// In TOML it is represented as a table with optional "foreground",
// "background", and "attributes" string fields.
//
// The original string values are stored so that Tag() can emit tview color
// tags without lossy named-color → hex round-tripping.
type StyleWrapper struct {
	tcell.Style
	fgStr   string // original foreground string from TOML / makeStyle
	bgStr   string // original background string
	attrStr string // tview attribute chars, e.g. "b", "du"
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
		s.fgStr = fg
	}
	if bg, ok := m["background"].(string); ok && bg != "" {
		style = style.Background(tcell.GetColor(bg))
		s.bgStr = bg
	}
	if attrs, ok := m["attributes"].(string); ok && attrs != "" {
		mask, err := stringToAttrMask(attrs)
		if err != nil {
			return err
		}
		style = style.Attributes(mask)
		s.attrStr = attrsToTviewString(attrs)
	}

	s.Style = style
	return nil
}

// Tag returns the tview inline color tag for this style, e.g. "[green::b]".
// Empty components use "-" (tview's "keep current" marker).
func (s StyleWrapper) Tag() string {
	fg := s.fgStr
	if fg == "" {
		fg = "-"
	}
	bg := s.bgStr
	if bg == "" {
		bg = "-"
	}
	attr := s.attrStr
	if attr == "" {
		attr = "-"
	}
	// Omit bg and attr when they are both default to keep tags short.
	if bg == "-" && attr == "-" {
		return "[" + fg + "]"
	}
	return "[" + fg + ":" + bg + ":" + attr + "]"
}

// Reset returns the tview tag that resets all inline styles.
func (s StyleWrapper) Reset() string {
	// Build a reset that undoes exactly what Tag() set.
	if s.bgStr == "" && s.attrStr == "" {
		return "[-]"
	}
	return "[-::-]"
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

// attrsToTviewString converts a pipe-separated attribute string (e.g.
// "bold|underline") to the compact tview format (e.g. "bu").
func attrsToTviewString(attrs string) string {
	var b strings.Builder
	for _, part := range strings.Split(attrs, "|") {
		part = strings.TrimSpace(strings.ToLower(part))
		switch part {
		case "bold":
			b.WriteByte('b')
		case "italic":
			b.WriteByte('i')
		case "underline":
			b.WriteByte('u')
		case "dim":
			b.WriteByte('d')
		case "reverse":
			b.WriteByte('r')
		case "blink":
			b.WriteByte('l')
		case "strikethrough":
			b.WriteByte('s')
		}
	}
	return b.String()
}

// makeStyle constructs a StyleWrapper from string arguments.
// fg and bg are tview/tcell color names or hex codes; attrs uses tview's
// compact format (e.g. "b" for bold, "du" for dim+underline).
func makeStyle(fg, bg, attrs string) StyleWrapper {
	style := tcell.StyleDefault
	if fg != "" {
		style = style.Foreground(tcell.GetColor(fg))
	}
	if bg != "" {
		style = style.Background(tcell.GetColor(bg))
	}
	if attrs != "" {
		var mask tcell.AttrMask
		for _, ch := range attrs {
			switch ch {
			case 'b':
				mask |= tcell.AttrBold
			case 'i':
				mask |= tcell.AttrItalic
			case 'u':
				mask |= tcell.AttrUnderline
			case 'd':
				mask |= tcell.AttrDim
			case 'r':
				mask |= tcell.AttrReverse
			case 'l':
				mask |= tcell.AttrBlink
			case 's':
				mask |= tcell.AttrStrikeThrough
			}
		}
		style = style.Attributes(mask)
	}
	return StyleWrapper{
		Style:   style,
		fgStr:   fg,
		bgStr:   bg,
		attrStr: attrs,
	}
}

// Foreground returns the foreground tcell.Color of this style.
func (s StyleWrapper) Foreground() tcell.Color {
	fg, _, _ := s.Style.Decompose()
	return fg
}

// Background returns the background tcell.Color of this style.
func (s StyleWrapper) Background() tcell.Color {
	_, bg, _ := s.Style.Decompose()
	return bg
}

// Theme holds the complete theme configuration.
type Theme struct {
	Preset       string            `toml:"preset"`
	Border       BorderTheme       `toml:"border"`
	Title        TitleTheme        `toml:"title"`
	ChannelsTree ChannelsTreeTheme `toml:"channels_tree"`
	MessagesList MessagesListTheme `toml:"messages_list"`
	MessageInput MessageInputTheme `toml:"message_input"`
	ThreadView   ThreadViewTheme   `toml:"thread_view"`
	Markdown     MarkdownTheme     `toml:"markdown_style"`
	Modal        ModalTheme        `toml:"modal"`
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
	Message          StyleWrapper `toml:"message"`
	Author           StyleWrapper `toml:"author"`
	Timestamp        StyleWrapper `toml:"timestamp"`
	Selected         StyleWrapper `toml:"selected"`
	Reply            StyleWrapper `toml:"reply"`
	SystemMessage    StyleWrapper `toml:"system_message"`
	EditedIndicator  StyleWrapper `toml:"edited_indicator"`
	PinIndicator     StyleWrapper `toml:"pin_indicator"`
	FileAttachment   StyleWrapper `toml:"file_attachment"`
	AttachmentTitle  StyleWrapper `toml:"attachment_title"`
	AttachmentText   StyleWrapper `toml:"attachment_text"`
	AttachmentFooter StyleWrapper `toml:"attachment_footer"`
	ReactionSelf     StyleWrapper `toml:"reaction_self"`
	ReactionOther    StyleWrapper `toml:"reaction_other"`
	DateSeparator    StyleWrapper `toml:"date_separator"`
	NewMsgSeparator  StyleWrapper `toml:"new_msg_separator"`
}

// MessageInputTheme configures the message input styling.
type MessageInputTheme struct {
	Text        StyleWrapper `toml:"text"`
	Placeholder StyleWrapper `toml:"placeholder"`
}

// ThreadViewTheme configures the thread view styling.
type ThreadViewTheme struct {
	Author           StyleWrapper `toml:"author"`
	Timestamp        StyleWrapper `toml:"timestamp"`
	ParentLabel      StyleWrapper `toml:"parent_label"`
	Separator        StyleWrapper `toml:"separator"`
	EditedIndicator  StyleWrapper `toml:"edited_indicator"`
	FileAttachment   StyleWrapper `toml:"file_attachment"`
	AttachmentTitle  StyleWrapper `toml:"attachment_title"`
	AttachmentText   StyleWrapper `toml:"attachment_text"`
	AttachmentFooter StyleWrapper `toml:"attachment_footer"`
	Reaction         StyleWrapper `toml:"reaction"`
}

// MarkdownTheme configures markdown rendering colors.
type MarkdownTheme struct {
	UserMention    StyleWrapper `toml:"user_mention"`
	ChannelMention StyleWrapper `toml:"channel_mention"`
	SpecialMention StyleWrapper `toml:"special_mention"`
	Link           StyleWrapper `toml:"link"`
	InlineCode     StyleWrapper `toml:"inline_code"`
	CodeFence      StyleWrapper `toml:"code_fence"`
	BlockquoteMark StyleWrapper `toml:"blockquote_mark"`
	BlockquoteText StyleWrapper `toml:"blockquote_text"`
}

// ModalTheme configures modal popup styling.
type ModalTheme struct {
	InputBackground StyleWrapper `toml:"input_background"`
	SecondaryText   StyleWrapper `toml:"secondary_text"`
}

// StatusBarTheme configures the status bar styling.
type StatusBarTheme struct {
	Text       StyleWrapper `toml:"text"`
	Background StyleWrapper `toml:"background"`
}
