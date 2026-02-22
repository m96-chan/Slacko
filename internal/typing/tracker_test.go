package typing

import (
	"testing"
	"time"
)

func TestFormatStatus_Empty(t *testing.T) {
	tr := NewTracker(nil)
	if got := tr.FormatStatus("C1"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFormatStatus_OneUser(t *testing.T) {
	tr := NewTracker(nil)
	tr.Add("C1", "U1", "alice")
	want := "alice is typing..."
	if got := tr.FormatStatus("C1"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatStatus_TwoUsers(t *testing.T) {
	tr := NewTracker(nil)
	tr.Add("C1", "U1", "alice")
	tr.Add("C1", "U2", "bob")
	want := "alice and bob are typing..."
	if got := tr.FormatStatus("C1"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatStatus_ThreeUsers(t *testing.T) {
	tr := NewTracker(nil)
	tr.Add("C1", "U1", "alice")
	tr.Add("C1", "U2", "bob")
	tr.Add("C1", "U3", "charlie")
	want := "alice, bob, and 1 other are typing..."
	if got := tr.FormatStatus("C1"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatStatus_FourUsers(t *testing.T) {
	tr := NewTracker(nil)
	tr.Add("C1", "U1", "alice")
	tr.Add("C1", "U2", "bob")
	tr.Add("C1", "U3", "charlie")
	tr.Add("C1", "U4", "dave")
	want := "alice, bob, and 2 others are typing..."
	if got := tr.FormatStatus("C1"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatStatus_DifferentChannels(t *testing.T) {
	tr := NewTracker(nil)
	tr.Add("C1", "U1", "alice")
	tr.Add("C2", "U2", "bob")
	if got := tr.FormatStatus("C1"); got != "alice is typing..." {
		t.Errorf("C1: got %q", got)
	}
	if got := tr.FormatStatus("C2"); got != "bob is typing..." {
		t.Errorf("C2: got %q", got)
	}
}

func TestClear(t *testing.T) {
	tr := NewTracker(nil)
	tr.Add("C1", "U1", "alice")
	tr.Clear("C1")
	if got := tr.FormatStatus("C1"); got != "" {
		t.Errorf("expected empty after clear, got %q", got)
	}
}

func TestOnChangeCallback(t *testing.T) {
	called := make(chan string, 10)
	tr := NewTracker(func(channelID string) {
		called <- channelID
	})
	tr.Add("C1", "U1", "alice")

	select {
	case ch := <-called:
		if ch != "C1" {
			t.Errorf("expected C1, got %s", ch)
		}
	case <-time.After(time.Second):
		t.Error("onChange not called")
	}
}

func TestRefreshDoesNotDuplicate(t *testing.T) {
	tr := NewTracker(nil)
	tr.Add("C1", "U1", "alice")
	tr.Add("C1", "U1", "alice") // refresh
	want := "alice is typing..."
	if got := tr.FormatStatus("C1"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
