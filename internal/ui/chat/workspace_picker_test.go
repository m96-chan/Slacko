package chat

import (
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func TestNewWorkspacePicker(t *testing.T) {
	wp := NewWorkspacePicker(&config.Config{})
	if wp == nil {
		t.Fatal("NewWorkspacePicker returned nil")
	}
}

func TestWorkspacePickerSetWorkspaces(t *testing.T) {
	wp := NewWorkspacePicker(&config.Config{})
	wp.SetCurrentWorkspace("team-1")
	wp.SetWorkspaces([]WorkspaceEntry{
		{ID: "team-1", Name: "Alpha"},
		{ID: "team-2", Name: "Beta"},
	})

	if wp.list.GetItemCount() != 2 {
		t.Fatalf("list count = %d, want 2", wp.list.GetItemCount())
	}

	main1, _ := wp.list.GetItemText(0)
	if main1 != "Alpha (current)" {
		t.Errorf("item 0 = %q, want %q", main1, "Alpha (current)")
	}

	main2, _ := wp.list.GetItemText(1)
	if main2 != "Beta" {
		t.Errorf("item 1 = %q, want %q", main2, "Beta")
	}
}

func TestWorkspacePickerSetWorkspacesEmpty(t *testing.T) {
	wp := NewWorkspacePicker(&config.Config{})
	wp.SetWorkspaces(nil)

	if wp.list.GetItemCount() != 0 {
		t.Errorf("list count = %d, want 0", wp.list.GetItemCount())
	}
	got := wp.status.GetText(false)
	if got != " No workspaces configured" {
		t.Errorf("status = %q, want %q", got, " No workspaces configured")
	}
}

func TestWorkspacePickerSetStatus(t *testing.T) {
	wp := NewWorkspacePicker(&config.Config{})
	wp.SetStatus("Switching...")
	got := wp.status.GetText(false)
	if got != " Switching..." {
		t.Errorf("status = %q, want %q", got, " Switching...")
	}
}

func TestWorkspacePickerReset(t *testing.T) {
	wp := NewWorkspacePicker(&config.Config{})
	wp.SetWorkspaces([]WorkspaceEntry{
		{ID: "team-1", Name: "Alpha"},
		{ID: "team-2", Name: "Beta"},
	})
	wp.list.SetCurrentItem(1)
	wp.Reset()

	if wp.list.GetCurrentItem() != 0 {
		t.Errorf("current item = %d, want 0", wp.list.GetCurrentItem())
	}
}

func TestWorkspacePickerCallbacks(t *testing.T) {
	wp := NewWorkspacePicker(&config.Config{})

	var selectCalled string
	closeCalled := false

	wp.SetOnSelect(func(id string) { selectCalled = id })
	wp.SetOnClose(func() { closeCalled = true })

	if wp.onSelect == nil {
		t.Error("onSelect not set")
	}
	if wp.onClose == nil {
		t.Error("onClose not set")
	}

	// Trigger close callback directly.
	wp.onClose()
	if !closeCalled {
		t.Error("onClose not called")
	}

	wp.onSelect("team-3")
	if selectCalled != "team-3" {
		t.Errorf("onSelect got %q, want %q", selectCalled, "team-3")
	}
}
