package main

import (
	"log/slog"
	"os"

	"github.com/m96-chan/Slacko/cmd"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := cmd.Run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}
