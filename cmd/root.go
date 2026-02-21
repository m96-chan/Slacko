package cmd

import (
	"flag"
	"log/slog"

	"github.com/m96-chan/Slacko/internal/app"
	"github.com/m96-chan/Slacko/internal/config"
	"github.com/m96-chan/Slacko/internal/logger"
)

// Run parses CLI flags, sets up logging and config, and starts the app.
func Run() error {
	configPath := flag.String("config-path", config.DefaultPath(), "path to config file")
	logPath := flag.String("log-path", logger.DefaultPath(), "path to log file")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

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
