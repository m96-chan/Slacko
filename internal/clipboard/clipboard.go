package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
)

// provider caches the detected clipboard provider on first use.
var (
	providerOnce sync.Once
	copyCmd      []string
	pasteCmd     []string
)

func detectProvider() {
	providerOnce.Do(func() {
		switch runtime.GOOS {
		case "darwin":
			copyCmd = []string{"pbcopy"}
			pasteCmd = []string{"pbpaste"}
		case "windows":
			copyCmd = []string{"clip.exe"}
			pasteCmd = []string{"powershell.exe", "-NoProfile", "-Command", "Get-Clipboard"}
		default: // Linux, FreeBSD, etc.
			// Prefer wl-copy/wl-paste for Wayland, fall back to xclip, then xsel.
			switch {
			case hasCommand("wl-copy"):
				copyCmd = []string{"wl-copy"}
				pasteCmd = []string{"wl-paste", "--no-newline"}
			case hasCommand("xclip"):
				copyCmd = []string{"xclip", "-selection", "clipboard"}
				pasteCmd = []string{"xclip", "-selection", "clipboard", "-o"}
			case hasCommand("xsel"):
				copyCmd = []string{"xsel", "--clipboard", "--input"}
				pasteCmd = []string{"xsel", "--clipboard", "--output"}
			}
		}
	})
}

// hasCommand checks if a command is available in PATH.
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Available returns true if a clipboard provider was detected.
func Available() bool {
	detectProvider()
	return len(copyCmd) > 0 && len(pasteCmd) > 0
}

// WriteText copies text to the system clipboard.
func WriteText(text string) error {
	detectProvider()
	if len(copyCmd) == 0 {
		return fmt.Errorf("clipboard: no copy command available")
	}

	cmd := exec.Command(copyCmd[0], copyCmd[1:]...)
	cmd.Stdin = bytes.NewReader([]byte(text))
	return cmd.Run()
}

// ReadText returns text from the system clipboard.
func ReadText() (string, error) {
	detectProvider()
	if len(pasteCmd) == 0 {
		return "", fmt.Errorf("clipboard: no paste command available")
	}

	cmd := exec.Command(pasteCmd[0], pasteCmd[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("clipboard: paste failed: %w", err)
	}
	return string(out), nil
}
