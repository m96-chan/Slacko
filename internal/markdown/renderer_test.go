package markdown

import (
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

var testUsers = map[string]slack.User{
	"U1": {ID: "U1", Name: "alice", Profile: slack.UserProfile{DisplayName: "Alice"}},
	"U2": {ID: "U2", Name: "bob", RealName: "Bob Jones"},
}

var testChannels = map[string]string{
	"C1": "general",
	"C2": "random",
}

var defColors = DefaultMarkdownColors()

func TestRender_Disabled(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"plain text", "hello world", "hello world"},
		{"user mention", "hi <@U1>", "hi @Alice"},
		{"user mention with label", "hi <@U1|alice>", "hi @alice"},
		{"unknown user", "hi <@U999>", "hi @U999"},
		{"channel mention", "see <#C1>", "see #general"},
		{"channel mention with label", "see <#C1|general>", "see #general"},
		{"special mention here", "<!here>", "@here"},
		{"special mention channel", "<!channel>", "@channel"},
		{"special mention everyone", "<!everyone>", "@everyone"},
		{"link with label", "<https://example.com|Example>", "Example"},
		{"link without label", "<https://example.com>", "https://example.com"},
		{"formatting not applied", "*bold* _italic_ ~strike~", "*bold* _italic_ ~strike~"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.text, testUsers, testChannels, false, "", defColors)
			if got != tt.want {
				t.Errorf("Render(%q, disabled) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestRender_Bold(t *testing.T) {
	got := Render("hello *world*", nil, nil, true, "", defColors)
	want := "hello [::b]world[::-]"
	if got != want {
		t.Errorf("bold: got %q, want %q", got, want)
	}
}

func TestRender_Italic(t *testing.T) {
	got := Render("hello _world_", nil, nil, true, "", defColors)
	want := "hello [::i]world[::-]"
	if got != want {
		t.Errorf("italic: got %q, want %q", got, want)
	}
}

func TestRender_Strikethrough(t *testing.T) {
	got := Render("hello ~world~", nil, nil, true, "", defColors)
	want := "hello [::s]world[::-]"
	if got != want {
		t.Errorf("strikethrough: got %q, want %q", got, want)
	}
}

func TestRender_InlineCode(t *testing.T) {
	got := Render("use `fmt.Println`", nil, nil, true, "", defColors)
	want := "use [gray]`fmt.Println`[-]"
	if got != want {
		t.Errorf("inline code: got %q, want %q", got, want)
	}
}

func TestRender_InlineCode_NoFormattingInside(t *testing.T) {
	got := Render("`*not bold*`", nil, nil, true, "", defColors)
	// The inline code regex matches first, so *not bold* should be inside code style
	// and not processed by bold regex.
	if strings.Contains(got, "[::b]") {
		t.Errorf("bold should not be applied inside inline code: got %q", got)
	}
	if !strings.Contains(got, "[gray]") {
		t.Errorf("inline code style should be present: got %q", got)
	}
}

func TestRender_UserMention(t *testing.T) {
	got := Render("hi <@U1>!", testUsers, nil, true, "", defColors)
	want := "hi [yellow::b]@Alice[-::-]!"
	if got != want {
		t.Errorf("user mention: got %q, want %q", got, want)
	}
}

func TestRender_UserMention_WithLabel(t *testing.T) {
	got := Render("hi <@U1|alice>!", testUsers, nil, true, "", defColors)
	want := "hi [yellow::b]@alice[-::-]!"
	if got != want {
		t.Errorf("user mention with label: got %q, want %q", got, want)
	}
}

func TestRender_UserMention_Unknown(t *testing.T) {
	got := Render("hi <@U999>!", testUsers, nil, true, "", defColors)
	want := "hi [yellow::b]@U999[-::-]!"
	if got != want {
		t.Errorf("unknown user mention: got %q, want %q", got, want)
	}
}

func TestRender_ChannelMention(t *testing.T) {
	got := Render("see <#C1>", nil, testChannels, true, "", defColors)
	want := "see [cyan::b]#general[-::-]"
	if got != want {
		t.Errorf("channel mention: got %q, want %q", got, want)
	}
}

func TestRender_ChannelMention_WithLabel(t *testing.T) {
	got := Render("see <#C1|general>", nil, testChannels, true, "", defColors)
	want := "see [cyan::b]#general[-::-]"
	if got != want {
		t.Errorf("channel mention with label: got %q, want %q", got, want)
	}
}

func TestRender_SpecialMention(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"here", "<!here>", "[yellow::bu]@here[-::-]"},
		{"channel", "<!channel>", "[yellow::bu]@channel[-::-]"},
		{"everyone", "<!everyone>", "[yellow::bu]@everyone[-::-]"},
		{"with label", "<!here|here>", "[yellow::bu]here[-::-]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.text, nil, nil, true, "", defColors)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRender_Link(t *testing.T) {
	got := Render("<https://example.com|Click here>", nil, nil, true, "", defColors)
	want := "[blue::u]Click here[-::-]"
	if got != want {
		t.Errorf("link with label: got %q, want %q", got, want)
	}
}

func TestRender_Link_NoLabel(t *testing.T) {
	got := Render("<https://example.com>", nil, nil, true, "", defColors)
	want := "[blue::u]https://example.com[-::-]"
	if got != want {
		t.Errorf("link without label: got %q, want %q", got, want)
	}
}

func TestRender_Blockquote(t *testing.T) {
	got := Render("> quoted text", nil, nil, true, "", defColors)
	want := "[gray]▎[-] [::d]quoted text[::-]"
	if got != want {
		t.Errorf("blockquote: got %q, want %q", got, want)
	}
}

func TestRender_Blockquote_Empty(t *testing.T) {
	got := Render(">", nil, nil, true, "", defColors)
	want := "[gray]▎[-]"
	if got != want {
		t.Errorf("empty blockquote: got %q, want %q", got, want)
	}
}

func TestRender_Emoji(t *testing.T) {
	got := Render("nice :thumbsup:", nil, nil, true, "", defColors)
	want := "nice 👍"
	if got != want {
		t.Errorf("emoji: got %q, want %q", got, want)
	}
}

func TestRender_Emoji_Unknown(t *testing.T) {
	got := Render("nice :custom_emoji:", nil, nil, true, "", defColors)
	want := "nice :custom_emoji:"
	if got != want {
		t.Errorf("unknown emoji: got %q, want %q", got, want)
	}
}

func TestRender_CodeBlock(t *testing.T) {
	text := "```\nfmt.Println(\"hello\")\n```"
	got := Render(text, nil, nil, true, "monokai", defColors)

	if !strings.Contains(got, "[gray]```[-]") {
		t.Errorf("code block should have styled fences: got %q", got)
	}
	if !strings.Contains(got, "Println") {
		t.Errorf("code block should contain code content: got %q", got)
	}
}

func TestRender_CodeBlock_WithLang(t *testing.T) {
	text := "```go\npackage main\n```"
	got := Render(text, nil, nil, true, "monokai", defColors)

	if !strings.Contains(got, "[gray]```[-]") {
		t.Errorf("code block should have styled fences: got %q", got)
	}
	if !strings.Contains(got, "go") || !strings.Contains(got, "package") {
		t.Errorf("code block should contain lang and code: got %q", got)
	}
}

func TestRender_Mixed(t *testing.T) {
	text := "Hey <@U1>, check *this* out :fire:"
	got := Render(text, testUsers, nil, true, "", defColors)

	if !strings.Contains(got, "[yellow::b]@Alice[-::-]") {
		t.Errorf("should contain styled user mention: got %q", got)
	}
	if !strings.Contains(got, "[::b]this[::-]") {
		t.Errorf("should contain bold text: got %q", got)
	}
	if !strings.Contains(got, "🔥") {
		t.Errorf("should contain fire emoji: got %q", got)
	}
}

func TestRender_TviewEscape(t *testing.T) {
	// Text with tview color tag syntax should be escaped.
	got := Render("[red]not a color tag[-]", nil, nil, true, "", defColors)
	// tview.Escape converts [ followed by content ] into escaped form.
	if strings.Contains(got, "[red]") && !strings.Contains(got, "[]") {
		t.Errorf("tview tags in user text should be escaped: got %q", got)
	}
}

func TestRender_MultipleLines(t *testing.T) {
	text := "*bold line*\n_italic line_"
	got := Render(text, nil, nil, true, "", defColors)

	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Errorf("should have 2 lines, got %d: %q", len(lines), got)
	}
	if !strings.Contains(lines[0], "[::b]") {
		t.Errorf("first line should be bold: %q", lines[0])
	}
	if !strings.Contains(lines[1], "[::i]") {
		t.Errorf("second line should be italic: %q", lines[1])
	}
}

func TestSplitCodeBlocks(t *testing.T) {
	text := "before\n```go\ncode\n```\nafter"
	segs := splitCodeBlocks(text)

	if len(segs) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segs))
	}
	if segs[0].isCode || segs[0].text != "before\n" {
		t.Errorf("seg 0: expected inline 'before\\n', got %+v", segs[0])
	}
	if !segs[1].isCode || segs[1].lang != "go" || segs[1].code != "code\n" {
		t.Errorf("seg 1: expected code block, got %+v", segs[1])
	}
	if segs[2].isCode || segs[2].text != "\nafter" {
		t.Errorf("seg 2: expected inline '\\nafter', got %+v", segs[2])
	}
}

func TestSplitCodeBlocks_NoBlocks(t *testing.T) {
	segs := splitCodeBlocks("just text")
	if len(segs) != 1 || segs[0].isCode {
		t.Errorf("expected 1 inline segment, got %+v", segs)
	}
}

func TestLookupEmoji(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"thumbsup", "👍"},
		{"+1", "👍"},
		{"heart", "❤️"},
		{"fire", "🔥"},
		{"unknown_custom", ":unknown_custom:"},
		// shortcodes not in the old hand-maintained map
		{"avocado", "🥑"},
		{"unicorn", "🦄"},
		{"pretzel", "🥨"},
		{"lobster", "🦞"},
		{"mango", "🥭"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lookupEmoji(tt.name)
			if got != tt.want {
				t.Errorf("lookupEmoji(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestRender_Blockquote_MultiLine(t *testing.T) {
	text := "> line 1\n> line 2\nnormal"
	got := Render(text, nil, nil, true, "", defColors)

	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), got)
	}
	if !strings.Contains(lines[0], "[gray]▎[-]") {
		t.Errorf("line 0 should be blockquote: %q", lines[0])
	}
	if !strings.Contains(lines[1], "[gray]▎[-]") {
		t.Errorf("line 1 should be blockquote: %q", lines[1])
	}
	if strings.Contains(lines[2], "▎") {
		t.Errorf("line 2 should not be blockquote: %q", lines[2])
	}
}

func TestRender_EmptyText(t *testing.T) {
	got := Render("", nil, nil, true, "", defColors)
	if got != "" {
		t.Errorf("empty text should return empty: got %q", got)
	}
}

func TestRender_UserMention_FallbackToName(t *testing.T) {
	users := map[string]slack.User{
		"U3": {ID: "U3", Name: "charlie"},
	}
	got := Render("<@U3>", users, nil, true, "", defColors)
	want := "[yellow::b]@charlie[-::-]"
	if got != want {
		t.Errorf("mention fallback to Name: got %q, want %q", got, want)
	}
}
