package cmd

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/m96-chan/Slacko/internal/app"
	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/logger"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Run parses CLI flags, sets up logging and config, and starts the app.
func Run() error {
	showVersion := flag.Bool("version", false, "print version and exit")
	configPath := flag.String("config-path", config.DefaultPath(), "path to config file")
	logPath := flag.String("log-path", logger.DefaultPath(), "path to log file")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("slacko %s (commit: %s, built: %s)\n", Version, Commit, Date)
		os.Exit(0)
	}

	level := parseLevel(*logLevel)
	if err := logger.Setup(*logPath, level); err != nil {
		return err
	}

	slog.Info("starting slacko", "config", *configPath, "log", *logPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	return app.New(cfg).Run()
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
