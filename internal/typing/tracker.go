package typing

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const expiryDuration = 5 * time.Second

// entry tracks a single user who is currently typing.
type entry struct {
	UserName string
	Expiry   time.Time
}

// Tracker tracks which users are currently typing in each channel.
// It is safe for concurrent use.
type Tracker struct {
	mu       sync.Mutex
	channels map[string]map[string]*entry // channelID → userID → entry
	onChange func(channelID string)
}

// NewTracker creates a new Tracker. The onChange callback is called (from a
// background goroutine) whenever the typing state changes for a channel.
func NewTracker(onChange func(channelID string)) *Tracker {
	return &Tracker{
		channels: make(map[string]map[string]*entry),
		onChange: onChange,
	}
}

// Add registers or refreshes a typing user. The indicator will auto-expire
// after 5 seconds unless refreshed.
func (t *Tracker) Add(channelID, userID, userName string) {
	t.mu.Lock()
	if t.channels[channelID] == nil {
		t.channels[channelID] = make(map[string]*entry)
	}
	isNew := t.channels[channelID][userID] == nil
	t.channels[channelID][userID] = &entry{
		UserName: userName,
		Expiry:   time.Now().Add(expiryDuration),
	}
	t.mu.Unlock()

	if isNew && t.onChange != nil {
		t.onChange(channelID)
	}

	// Schedule expiry cleanup.
	go func() {
		time.Sleep(expiryDuration)
		t.expire(channelID, userID)
	}()
}

// expire removes a typing entry if it has not been refreshed.
func (t *Tracker) expire(channelID, userID string) {
	t.mu.Lock()
	users := t.channels[channelID]
	if users == nil {
		t.mu.Unlock()
		return
	}
	e, ok := users[userID]
	if !ok {
		t.mu.Unlock()
		return
	}
	if time.Now().Before(e.Expiry) {
		// Entry was refreshed; don't remove.
		t.mu.Unlock()
		return
	}
	delete(users, userID)
	if len(users) == 0 {
		delete(t.channels, channelID)
	}
	t.mu.Unlock()

	if t.onChange != nil {
		t.onChange(channelID)
	}
}

// Clear removes all typing entries for a channel.
func (t *Tracker) Clear(channelID string) {
	t.mu.Lock()
	delete(t.channels, channelID)
	t.mu.Unlock()
}

// FormatStatus returns a formatted typing status string for a channel.
// Returns empty string if no one is typing.
//
// Patterns:
//   - "alice is typing..."
//   - "alice and bob are typing..."
//   - "alice, bob, and 2 others are typing..."
func (t *Tracker) FormatStatus(channelID string) string {
	t.mu.Lock()
	users := t.channels[channelID]
	if len(users) == 0 {
		t.mu.Unlock()
		return ""
	}

	// Collect non-expired usernames.
	now := time.Now()
	names := make([]string, 0, len(users))
	for _, e := range users {
		if now.Before(e.Expiry) {
			names = append(names, e.UserName)
		}
	}
	t.mu.Unlock()

	if len(names) == 0 {
		return ""
	}

	sort.Strings(names)

	switch len(names) {
	case 1:
		return fmt.Sprintf("%s is typing...", names[0])
	case 2:
		return fmt.Sprintf("%s and %s are typing...", names[0], names[1])
	default:
		first := strings.Join(names[:2], ", ")
		others := len(names) - 2
		if others == 1 {
			return fmt.Sprintf("%s, and 1 other are typing...", first)
		}
		return fmt.Sprintf("%s, and %d others are typing...", first, others)
	}
}
