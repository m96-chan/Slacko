package logger

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/m96-chan/Slacko/internal/consts"
)

// DefaultPath returns the default log file path inside the cache directory.
func DefaultPath() string {
	return filepath.Join(consts.CacheDir, consts.Name+".log")
}

// Setup configures the global slog logger to write to the given file path
// at the specified level.
func Setup(path string, level slog.Level) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}

	handler := slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
	return nil
}
