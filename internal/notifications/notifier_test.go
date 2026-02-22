package notifications

import (
	"testing"
	"time"
)

func TestDetectMention(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		selfUserID string
		isDM       bool
		want       MentionType
	}{
		{"DM", "hello", "U1", true, MentionDM},
		{"direct mention", "hey <@U1> check this", "U1", false, MentionDirect},
		{"direct mention with label", "hey <@U1|alice> check this", "U1", false, MentionDirect},
		{"everyone", "<!everyone> heads up", "U1", false, MentionEveryone},
		{"channel", "<!channel> important", "U1", false, MentionChannel},
		{"here", "<!here> anyone around?", "U1", false, MentionHere},
		{"no mention", "just a regular message", "U1", false, MentionNone},
		{"other user mention", "hey <@U2> check this", "U1", false, MentionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectMention(tt.text, tt.selfUserID, tt.isDM)
			if got != tt.want {
				t.Errorf("DetectMention(%q, %q, %v) = %d, want %d",
					tt.text, tt.selfUserID, tt.isDM, got, tt.want)
			}
		})
	}
}

func TestDetectMention_Priority(t *testing.T) {
	// DM takes priority over other mentions.
	got := DetectMention("<@U1> <!everyone>", "U1", true)
	if got != MentionDM {
		t.Errorf("DM should take priority, got %d", got)
	}

	// Direct mention takes priority over group mentions.
	got = DetectMention("<@U1> <!here>", "U1", false)
	if got != MentionDirect {
		t.Errorf("direct mention should take priority over here, got %d", got)
	}
}

func TestStripMrkdwn(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"bold", "*hello*", "hello"},
		{"italic", "_hello_", "hello"},
		{"strike", "~hello~", "hello"},
		{"code block", "```code```", "code"},
		{"inline code", "`code`", "code"},
		{"user mention with label", "<@U1|alice>", "@alice"},
		{"user mention", "<@U1>", "@U1"},
		{"channel mention", "<#C1|general>", "#general"},
		{"special mention", "<!here|here>", "here"},
		{"special mention no label", "<!here>", "@here"},
		{"link with label", "<https://example.com|Example>", "Example"},
		{"link without label", "<https://example.com>", "https://example.com"},
		{"no formatting", "plain text", "plain text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripMrkdwn(tt.text)
			if got != tt.want {
				t.Errorf("StripMrkdwn(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestStripMrkdwn_Truncation(t *testing.T) {
	long := ""
	for i := 0; i < 250; i++ {
		long += "a"
	}
	got := StripMrkdwn(long)
	if len(got) > 204 { // 200 + "…" (3 bytes in UTF-8) + possible trim
		t.Errorf("expected truncated output, got length %d", len(got))
	}
}

func TestNotifier_RateLimit(t *testing.T) {
	n := New()
	// First send should go through (sets lastSent).
	n.Send("title", "body")

	// Immediately after, lastSent should be recent.
	n.mu.Lock()
	elapsed := time.Since(n.lastSent)
	n.mu.Unlock()

	if elapsed > time.Second {
		t.Errorf("lastSent should be recent, elapsed %v", elapsed)
	}

	// A second Send right after should be rate-limited (lastSent not updated).
	firstSent := n.lastSent
	n.Send("title2", "body2")
	n.mu.Lock()
	secondSent := n.lastSent
	n.mu.Unlock()

	if !secondSent.Equal(firstSent) {
		t.Error("rate-limited Send should not update lastSent")
	}
}

func TestResolveToken(t *testing.T) {
	tests := []struct {
		name  string
		inner string
		want  string
	}{
		{"user with label", "@U1|alice", "@alice"},
		{"user without label", "@U1", "@U1"},
		{"channel with label", "#C1|general", "#general"},
		{"channel without label", "#C1", "#C1"},
		{"special with label", "!here|here", "here"},
		{"special without label", "!here", "@here"},
		{"url with label", "https://example.com|Example", "Example"},
		{"url without label", "https://example.com", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveToken(tt.inner)
			if got != tt.want {
				t.Errorf("resolveToken(%q) = %q, want %q", tt.inner, got, tt.want)
			}
		})
	}
}
