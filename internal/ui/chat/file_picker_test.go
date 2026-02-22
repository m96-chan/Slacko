package chat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m96-chan/Slacko/internal/config"
)

func newTestFilePicker(t *testing.T) *FilePicker {
	t.Helper()
	cfg := &config.Config{}
	cfg.Keybinds.FilePicker.Close = "Escape"
	cfg.Keybinds.FilePicker.Up = "Ctrl+P"
	cfg.Keybinds.FilePicker.Down = "Ctrl+N"
	cfg.Keybinds.FilePicker.Select = "Enter"
	return NewFilePicker(cfg)
}

func TestLoadDir(t *testing.T) {
	// Create a temp directory with known contents.
	tmp := t.TempDir()
	os.Mkdir(filepath.Join(tmp, "beta_dir"), 0o755)
	os.Mkdir(filepath.Join(tmp, "alpha_dir"), 0o755)
	os.WriteFile(filepath.Join(tmp, "file_b.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(tmp, "file_a.go"), []byte("package main"), 0o644)
	// Hidden file should be skipped.
	os.WriteFile(filepath.Join(tmp, ".hidden"), []byte("secret"), 0o644)

	fp := newTestFilePicker(t)
	fp.loadDir(tmp)

	if fp.currentDir != tmp {
		t.Errorf("currentDir = %q, want %q", fp.currentDir, tmp)
	}

	// Expected order: "..", alpha_dir, beta_dir, file_a.go, file_b.txt
	wantNames := []string{"..", "alpha_dir", "beta_dir", "file_a.go", "file_b.txt"}
	if len(fp.entries) != len(wantNames) {
		t.Fatalf("got %d entries, want %d: %v", len(fp.entries), len(wantNames), entryNames(fp.entries))
	}
	for i, want := range wantNames {
		if fp.entries[i].name != want {
			t.Errorf("entries[%d].name = %q, want %q", i, fp.entries[i].name, want)
		}
	}

	// Verify dirs are marked as dirs.
	if !fp.entries[0].isDir { // ..
		t.Error("'..' should be a directory")
	}
	if !fp.entries[1].isDir { // alpha_dir
		t.Error("alpha_dir should be a directory")
	}
	if fp.entries[3].isDir { // file_a.go
		t.Error("file_a.go should not be a directory")
	}

	// Verify hidden file is excluded.
	for _, e := range fp.entries {
		if e.name == ".hidden" {
			t.Error(".hidden should be excluded from entries")
		}
	}
}

func TestFileIcon(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"photo.jpg", "\U0001F5BC"},
		{"photo.PNG", "\U0001F5BC"},
		{"song.mp3", "\U0001F3B5"},
		{"movie.mp4", "\U0001F3AC"},
		{"archive.zip", "\U0001F4E6"},
		{"doc.pdf", "\U0001F4C4"},
		{"sheet.xlsx", "\U0001F4CA"},
		{"main.go", "\U0001F4DD"},
		{"readme.md", "\U0001F4DD"},
		{"data.csv", "\U0001F4CA"},
		{"unknown.xyz", "\U0001F4CE"},
		{"noext", "\U0001F4CE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileIcon(tt.name, false)
			if got != tt.want {
				t.Errorf("fileIcon(%q, false) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestNavigateUp(t *testing.T) {
	// Create nested temp directories.
	tmp := t.TempDir()
	child := filepath.Join(tmp, "child")
	os.Mkdir(child, 0o755)

	fp := newTestFilePicker(t)
	fp.loadDir(child)

	if fp.currentDir != child {
		t.Fatalf("currentDir = %q, want %q", fp.currentDir, child)
	}

	// First entry should be ".." pointing to parent.
	if len(fp.entries) == 0 {
		t.Fatal("expected at least '..' entry")
	}
	if fp.entries[0].name != ".." {
		t.Fatalf("entries[0].name = %q, want '..'", fp.entries[0].name)
	}
	if fp.entries[0].path != tmp {
		t.Errorf("'..' path = %q, want %q", fp.entries[0].path, tmp)
	}

	// Simulate selecting ".." — should navigate to parent.
	fp.list.SetCurrentItem(0)
	fp.selectCurrent()

	if fp.currentDir != tmp {
		t.Errorf("after navigating up, currentDir = %q, want %q", fp.currentDir, tmp)
	}
}

func TestSelectFile(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "test.txt")
	os.WriteFile(filePath, []byte("content"), 0o644)

	fp := newTestFilePicker(t)
	fp.loadDir(tmp)

	var selected string
	fp.SetOnSelect(func(path string) {
		selected = path
	})

	// Find the file entry and select it.
	for i, e := range fp.entries {
		if e.name == "test.txt" {
			fp.list.SetCurrentItem(i)
			break
		}
	}
	fp.selectCurrent()

	if selected != filePath {
		t.Errorf("selected = %q, want %q", selected, filePath)
	}
}

func TestSelectDirDescends(t *testing.T) {
	tmp := t.TempDir()
	subdir := filepath.Join(tmp, "subdir")
	os.Mkdir(subdir, 0o755)

	fp := newTestFilePicker(t)
	fp.loadDir(tmp)

	// Find the subdir entry and select it.
	for i, e := range fp.entries {
		if e.name == "subdir" {
			fp.list.SetCurrentItem(i)
			break
		}
	}
	fp.selectCurrent()

	if fp.currentDir != subdir {
		t.Errorf("currentDir = %q, want %q", fp.currentDir, subdir)
	}
}

func TestCloseCallback(t *testing.T) {
	fp := newTestFilePicker(t)

	closed := false
	fp.SetOnClose(func() {
		closed = true
	})

	fp.close()

	if !closed {
		t.Error("onClose callback should have been called")
	}
}

func entryNames(entries []fileEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
	}
	return names
}
