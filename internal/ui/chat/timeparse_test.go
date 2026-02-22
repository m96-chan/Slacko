package chat

import (
	"testing"
	"time"
)

func TestParseFutureTime_RelativeDuration(t *testing.T) {
	before := time.Now()
	got, remaining, err := ParseFutureTime("in 30m hello world")
	after := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != "hello world" {
		t.Errorf("remaining = %q, want %q", remaining, "hello world")
	}

	expected := before.Add(30 * time.Minute)
	if got.Before(expected.Add(-time.Second)) || got.After(after.Add(30*time.Minute).Add(time.Second)) {
		t.Errorf("time %v not within expected range around %v", got, expected)
	}
}

func TestParseFutureTime_RelativeHours(t *testing.T) {
	_, remaining, err := ParseFutureTime("in 2h reminder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != "reminder" {
		t.Errorf("remaining = %q, want %q", remaining, "reminder")
	}
}

func TestParseFutureTime_Tomorrow(t *testing.T) {
	got, remaining, err := ParseFutureTime("tomorrow 9am meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != "meeting" {
		t.Errorf("remaining = %q, want %q", remaining, "meeting")
	}
	if got.Hour() != 9 {
		t.Errorf("hour = %d, want 9", got.Hour())
	}

	tomorrow := time.Now().AddDate(0, 0, 1)
	if got.Day() != tomorrow.Day() {
		t.Errorf("day = %d, want %d", got.Day(), tomorrow.Day())
	}
}

func TestParseFutureTime_TomorrowPM(t *testing.T) {
	got, _, err := ParseFutureTime("tomorrow 3pm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hour() != 15 {
		t.Errorf("hour = %d, want 15", got.Hour())
	}
}

func TestParseFutureTime_Tomorrow24H(t *testing.T) {
	got, _, err := ParseFutureTime("tomorrow 14:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hour() != 14 || got.Minute() != 0 {
		t.Errorf("time = %02d:%02d, want 14:00", got.Hour(), got.Minute())
	}
}

func TestParseFutureTime_TomorrowDefault(t *testing.T) {
	got, _, err := ParseFutureTime("tomorrow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Hour() != 9 {
		t.Errorf("hour = %d, want 9 (default)", got.Hour())
	}
}

func TestParseFutureTime_ISO(t *testing.T) {
	got, remaining, err := ParseFutureTime("2026-03-01 10:00 hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != "hello" {
		t.Errorf("remaining = %q, want %q", remaining, "hello")
	}
	if got.Year() != 2026 || got.Month() != 3 || got.Day() != 1 {
		t.Errorf("date = %v, want 2026-03-01", got)
	}
	if got.Hour() != 10 || got.Minute() != 0 {
		t.Errorf("time = %02d:%02d, want 10:00", got.Hour(), got.Minute())
	}
}

func TestParseFutureTime_UnixTimestamp(t *testing.T) {
	ts := time.Now().Add(time.Hour).Unix()
	input := time.Now().Add(time.Hour).Format("1136239445") // just use the number
	_ = input

	got, _, err := ParseFutureTime("1800000000 message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = ts
	expected := time.Unix(1800000000, 0)
	if !got.Equal(expected) {
		t.Errorf("time = %v, want %v", got, expected)
	}
}

func TestParseFutureTime_Empty(t *testing.T) {
	_, _, err := ParseFutureTime("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseFutureTime_Unrecognized(t *testing.T) {
	_, _, err := ParseFutureTime("not a time")
	if err == nil {
		t.Error("expected error for unrecognized format")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		err   bool
	}{
		{"30m", 30 * time.Minute, false},
		{"1h", time.Hour, false},
		{"2h30m", 2*time.Hour + 30*time.Minute, false},
		{"90s", 90 * time.Second, false},
		{"5", 5 * time.Minute, false},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.err {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.err)
				return
			}
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		input    string
		wantHour int
		wantMin  int
		err      bool
	}{
		{"9am", 9, 0, false},
		{"12pm", 12, 0, false},
		{"12am", 0, 0, false},
		{"3pm", 15, 0, false},
		{"9:30am", 9, 30, false},
		{"14:00", 14, 0, false},
		{"", 9, 0, false}, // default
		{"invalid", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseTimeOfDay(tt.input)
			if (err != nil) != tt.err {
				t.Errorf("parseTimeOfDay(%q) error = %v, wantErr %v", tt.input, err, tt.err)
				return
			}
			if err == nil {
				if got.Hour() != tt.wantHour || got.Minute() != tt.wantMin {
					t.Errorf("parseTimeOfDay(%q) = %02d:%02d, want %02d:%02d",
						tt.input, got.Hour(), got.Minute(), tt.wantHour, tt.wantMin)
				}
			}
		})
	}
}

func TestSplitDurationAndMessage(t *testing.T) {
	tests := []struct {
		input   string
		wantDur string
		wantMsg string
	}{
		{"30m hello world", "30m", "hello world"},
		{"1h", "1h", ""},
		{"2h30m reminder", "2h30m", "reminder"},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			dur, msg := splitDurationAndMessage(tt.input)
			if dur != tt.wantDur {
				t.Errorf("dur = %q, want %q", dur, tt.wantDur)
			}
			if msg != tt.wantMsg {
				t.Errorf("msg = %q, want %q", msg, tt.wantMsg)
			}
		})
	}
}

func TestSplitFirstWord(t *testing.T) {
	tests := []struct {
		input     string
		wantFirst string
		wantRest  string
	}{
		{"hello world", "hello", "world"},
		{"single", "single", ""},
		{"  padded  text  ", "padded", "text"},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			first, rest := splitFirstWord(tt.input)
			if first != tt.wantFirst {
				t.Errorf("first = %q, want %q", first, tt.wantFirst)
			}
			if rest != tt.wantRest {
				t.Errorf("rest = %q, want %q", rest, tt.wantRest)
			}
		})
	}
}
