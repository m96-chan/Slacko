package chat

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseFutureTime parses a human-readable time string into a future time.Time.
// Supported formats:
//   - "in 30m", "in 1h", "in 2h30m" — relative duration
//   - "tomorrow 9am", "tomorrow 14:00" — next day
//   - "2026-03-01 10:00" — ISO date-time
//   - Unix timestamp (integer)
//
// Returns the parsed time, remaining unparsed text, and any error.
func ParseFutureTime(input string) (time.Time, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, "", fmt.Errorf("empty time string")
	}

	now := time.Now()

	// "in <duration>" format.
	if strings.HasPrefix(strings.ToLower(input), "in ") {
		rest := input[3:]
		// Find where the duration ends and the message begins.
		durStr, remaining := splitDurationAndMessage(rest)
		d, err := parseDuration(durStr)
		if err != nil {
			return time.Time{}, "", fmt.Errorf("invalid duration: %w", err)
		}
		return now.Add(d), remaining, nil
	}

	// "tomorrow [time]" format.
	lower := strings.ToLower(input)
	if strings.HasPrefix(lower, "tomorrow") {
		rest := strings.TrimSpace(input[8:])
		timeStr, remaining := splitFirstWord(rest)
		t, err := parseTimeOfDay(timeStr)
		if err != nil {
			return time.Time{}, "", fmt.Errorf("invalid time: %w", err)
		}
		tomorrow := now.AddDate(0, 0, 1)
		result := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
			t.Hour(), t.Minute(), 0, 0, now.Location())
		return result, remaining, nil
	}

	// ISO date-time: "2026-03-01 10:00 <message>"
	if len(input) >= 16 && input[4] == '-' && input[7] == '-' {
		dateTimeStr := input[:16]
		remaining := strings.TrimSpace(input[16:])
		t, err := time.ParseInLocation("2006-01-02 15:04", dateTimeStr, now.Location())
		if err != nil {
			return time.Time{}, "", fmt.Errorf("invalid date-time: %w", err)
		}
		return t, remaining, nil
	}

	// Unix timestamp.
	firstWord, remaining := splitFirstWord(input)
	if ts, err := strconv.ParseInt(firstWord, 10, 64); err == nil && ts > 1000000000 {
		return time.Unix(ts, 0), remaining, nil
	}

	return time.Time{}, "", fmt.Errorf("unrecognized time format: %s", input)
}

// parseDuration parses duration strings like "30m", "1h", "2h30m", "90s".
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	// Try Go's standard duration parser first.
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}
	// Try plain number as minutes.
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Minute, nil
	}
	return 0, fmt.Errorf("cannot parse %q as duration", s)
}

// parseTimeOfDay parses "9am", "14:00", "9:30pm" into a time with just hours/minutes.
func parseTimeOfDay(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		// Default to 9:00.
		return time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), nil
	}

	// Try "HH:MM" format.
	if t, err := time.Parse("15:04", s); err == nil {
		return t, nil
	}

	// Try "3pm", "9am" format.
	s = strings.TrimSpace(s)
	isPM := strings.HasSuffix(s, "pm")
	isAM := strings.HasSuffix(s, "am")
	if isPM || isAM {
		numStr := s[:len(s)-2]
		parts := strings.Split(numStr, ":")
		hour, err := strconv.Atoi(parts[0])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid hour: %s", numStr)
		}
		min := 0
		if len(parts) > 1 {
			min, err = strconv.Atoi(parts[1])
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid minute: %s", parts[1])
			}
		}
		if isPM && hour != 12 {
			hour += 12
		}
		if isAM && hour == 12 {
			hour = 0
		}
		return time.Date(0, 1, 1, hour, min, 0, 0, time.UTC), nil
	}

	return time.Time{}, fmt.Errorf("cannot parse %q as time of day", s)
}

// splitDurationAndMessage splits "30m hello world" into ("30m", "hello world").
// It looks for the first space after duration characters (digits, h, m, s).
func splitDurationAndMessage(s string) (string, string) {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) {
		c := s[i]
		if c == ' ' {
			break
		}
		i++
	}
	dur := s[:i]
	msg := strings.TrimSpace(s[i:])
	return dur, msg
}

// splitFirstWord splits "word rest of string" into ("word", "rest of string").
func splitFirstWord(s string) (string, string) {
	s = strings.TrimSpace(s)
	idx := strings.IndexByte(s, ' ')
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimSpace(s[idx+1:])
}
