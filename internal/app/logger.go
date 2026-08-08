package app

import (
	"fmt"
	"log/slog"
	"os"
)

const (
	defaultLogLevel   = "DEBUG"
	defaultFormatJSON = "json"
)

type loggerConfig struct {
	level  string
	format string
}

func (c *loggerConfig) applyDefaults() {
	if c.format == "" {
		c.format = defaultFormatJSON
	}

	if c.level == "" {
		c.level = defaultLogLevel
	}
}

var loggerWasInit = false

func setLogger(c *loggerConfig) error {
	c.applyDefaults()

	levelStr := fmt.Sprintf(`"%s"`, c.level)

	var l slog.Level
	if err := l.UnmarshalJSON([]byte(levelStr)); err != nil {
		return fmt.Errorf("cannot parse log level '%s': %w", c.level, err)
	}

	logWriter := os.Stderr
	logOpts := &slog.HandlerOptions{
		Level: l,
	}

	var handler slog.Handler

	switch c.format {
	case defaultFormatJSON:
		handler = slog.NewJSONHandler(logWriter, logOpts)
	case "text":
		handler = slog.NewTextHandler(logWriter, logOpts)
	default:
		return fmt.Errorf("incorrect log format '%s'", c.format)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return nil
}

func initLogger(c *loggerConfig) error {
	if loggerWasInit {
		return nil
	}

	if err := setLogger(c); err != nil {
		return err
	}

	loggerWasInit = true

	return nil
}
