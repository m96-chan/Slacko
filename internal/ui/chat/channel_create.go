package chat

import (
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/m96-chan/Slacko/internal/config"
)

// channelNameRe matches valid Slack channel names: lowercase letters, numbers,
// hyphens, and underscores. No spaces, no uppercase.
var channelNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ChannelCreateForm is a modal form for creating a new Slack channel.
type ChannelCreateForm struct {
	*tview.Flex
	cfg          *config.Config
	form         *tview.Form
	nameInput    *tview.InputField
	status       *tview.TextView
	privateCheck bool
	onCreate     func(name string, isPrivate bool)
	onClose      func()
}

// NewChannelCreateForm creates a new channel creation form.
func NewChannelCreateForm(cfg *config.Config) *ChannelCreateForm {
	f := &ChannelCreateForm{
		cfg: cfg,
	}

	f.nameInput = tview.NewInputField().
		SetLabel("Channel name").
		SetFieldWidth(40)
	f.nameInput.SetAcceptanceFunc(func(text string, lastChar rune) bool {
		// Only allow characters valid in Slack channel names.
		return lastChar == '-' || lastChar == '_' ||
			(lastChar >= 'a' && lastChar <= 'z') ||
			(lastChar >= '0' && lastChar <= '9')
	})

	f.form = tview.NewForm().
		AddFormItem(f.nameInput).
		AddCheckbox("Private channel", false, func(checked bool) {
			f.privateCheck = checked
		}).
		AddButton("Create", func() {
			f.submit()
		}).
		AddButton("Cancel", func() {
			if f.onClose != nil {
				f.onClose()
			}
		})
	f.form.SetBorder(true).SetTitle(" Create Channel ")
	f.form.SetInputCapture(f.handleInput)

	f.status = tview.NewTextView().SetDynamicColors(true)

	f.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(f.form, 0, 1, true).
		AddItem(f.status, 1, 0, false)

	return f
}

// SetOnCreate sets the callback invoked when the user submits the form.
func (f *ChannelCreateForm) SetOnCreate(fn func(name string, isPrivate bool)) {
	f.onCreate = fn
}

// SetOnClose sets the callback invoked when the form is dismissed.
func (f *ChannelCreateForm) SetOnClose(fn func()) {
	f.onClose = fn
}

// GetName returns the current channel name input value.
func (f *ChannelCreateForm) GetName() string {
	return strings.TrimSpace(f.nameInput.GetText())
}

// IsPrivate returns whether the private channel checkbox is checked.
func (f *ChannelCreateForm) IsPrivate() bool {
	return f.privateCheck
}

// SetStatus updates the status text at the bottom.
func (f *ChannelCreateForm) SetStatus(text string) {
	f.status.SetText(" " + text)
}

// Reset clears all form fields.
func (f *ChannelCreateForm) Reset() {
	f.nameInput.SetText("")
	f.privateCheck = false
	// Reset the checkbox in the form by re-creating it is complex in tview,
	// so we track the state separately via privateCheck.
	f.status.SetText("")
}

// validate checks that the channel name is valid per Slack's rules.
func (f *ChannelCreateForm) validate() bool {
	name := f.GetName()
	if name == "" {
		return false
	}
	return channelNameRe.MatchString(name)
}

// submit validates and triggers the onCreate callback.
func (f *ChannelCreateForm) submit() {
	if !f.validate() {
		f.SetStatus("Invalid name: use lowercase letters, numbers, hyphens, underscores")
		return
	}
	if f.onCreate != nil {
		f.onCreate(f.GetName(), f.privateCheck)
	}
}

// handleInput processes keybindings for the channel create form.
func (f *ChannelCreateForm) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		if f.onClose != nil {
			f.onClose()
		}
		return nil
	}
	return event
}
