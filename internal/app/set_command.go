package app

import (
	"fmt"
	"strings"

	"github.com/m96-chan/Slacko/internal/config"
)

// RuntimeOption defines a settable runtime option with typed getter/setter.
type RuntimeOption struct {
	Name string
	Get  func(*config.Config) bool
	Set  func(*config.Config, bool)
}

// runtimeOptions is the registry of all runtime-settable boolean options.
var runtimeOptions = []RuntimeOption{
	{
		Name: "mouse",
		Get:  func(c *config.Config) bool { return c.Mouse },
		Set:  func(c *config.Config, v bool) { c.Mouse = v },
	},
	{
		Name: "timestamps",
		Get:  func(c *config.Config) bool { return c.Timestamps.Enabled },
		Set:  func(c *config.Config, v bool) { c.Timestamps.Enabled = v },
	},
	{
		Name: "markdown",
		Get:  func(c *config.Config) bool { return c.Markdown.Enabled },
		Set:  func(c *config.Config, v bool) { c.Markdown.Enabled = v },
	},
	{
		Name: "typing",
		Get:  func(c *config.Config) bool { return c.TypingIndicator.Enabled },
		Set:  func(c *config.Config, v bool) { c.TypingIndicator.Enabled = v },
	},
	{
		Name: "presence",
		Get:  func(c *config.Config) bool { return c.Presence.Enabled },
		Set:  func(c *config.Config, v bool) { c.Presence.Enabled = v },
	},
	{
		Name: "date_separator",
		Get:  func(c *config.Config) bool { return c.DateSeparator.Enabled },
		Set:  func(c *config.Config, v bool) { c.DateSeparator.Enabled = v },
	},
}

// findOption looks up a runtime option by name.
func findOption(name string) (*RuntimeOption, bool) {
	for i := range runtimeOptions {
		if runtimeOptions[i].Name == name {
			return &runtimeOptions[i], true
		}
	}
	return nil, false
}

// boolString returns "on" or "off" for a boolean value.
func boolString(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

// ParseBoolValue parses a boolean value string.
// Accepted values: on/off, true/false, yes/no (case-insensitive).
func ParseBoolValue(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "on", "true", "yes":
		return true, nil
	case "off", "false", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value %q: use on/off, true/false, or yes/no", s)
	}
}

// ParseSetCommand parses the arguments to the :set command.
//
// Supported forms:
//   - "option=value"      — assign (option, value, false, nil)
//   - "option value"      — assign (option, value, false, nil)
//   - "option?"           — query  (option, "",    true,  nil)
//   - "option"            — toggle (option, "",    false, nil)
//   - ""                  — error
//
// The value is validated as a boolean (on/off, true/false, yes/no).
func ParseSetCommand(args string) (option, value string, query bool, err error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", false, fmt.Errorf("no option specified")
	}

	// Check for query form: "option?"
	if strings.HasSuffix(args, "?") {
		option = strings.TrimSpace(strings.TrimSuffix(args, "?"))
		return option, "", true, nil
	}

	// Check for "option=value" form.
	if idx := strings.Index(args, "="); idx >= 0 {
		option = strings.TrimSpace(args[:idx])
		value = strings.TrimSpace(args[idx+1:])
	} else if parts := strings.Fields(args); len(parts) == 2 {
		// "option value" form.
		option = parts[0]
		value = parts[1]
	} else if len(parts) == 1 {
		// Toggle form: just "option".
		return parts[0], "", false, nil
	} else {
		return "", "", false, fmt.Errorf("invalid syntax: %q", args)
	}

	// Validate value if provided.
	if value != "" {
		if _, err := ParseBoolValue(value); err != nil {
			return "", "", false, err
		}
	}

	return option, value, false, nil
}

// ApplySetCommand applies a :set command to the config.
//
// If value is empty, the option is toggled.
// If value is provided, it is set to the given boolean.
//
// Returns a human-readable feedback message.
func ApplySetCommand(cfg *config.Config, option, value string) (string, error) {
	opt, ok := findOption(option)
	if !ok {
		return "", fmt.Errorf("unknown option %q (available: %s)", option, strings.Join(RuntimeOptionNames(), ", "))
	}

	var newVal bool
	if value == "" {
		// Toggle.
		newVal = !opt.Get(cfg)
	} else {
		var err error
		newVal, err = ParseBoolValue(value)
		if err != nil {
			return "", err
		}
	}

	opt.Set(cfg, newVal)
	return fmt.Sprintf("%s = %s", option, boolString(newVal)), nil
}

// QueryOption returns the current value of a runtime option.
func QueryOption(cfg *config.Config, option string) (string, error) {
	opt, ok := findOption(option)
	if !ok {
		return "", fmt.Errorf("unknown option %q (available: %s)", option, strings.Join(RuntimeOptionNames(), ", "))
	}
	return fmt.Sprintf("%s = %s", option, boolString(opt.Get(cfg))), nil
}

// ListRuntimeOptions returns a formatted string of all settable options
// and their current values.
func ListRuntimeOptions(cfg *config.Config) string {
	var b strings.Builder
	b.WriteString("Runtime options:")
	for _, opt := range runtimeOptions {
		b.WriteString(fmt.Sprintf("  %s = %s", opt.Name, boolString(opt.Get(cfg))))
	}
	return b.String()
}

// RuntimeOptionNames returns the names of all runtime-settable options.
func RuntimeOptionNames() []string {
	names := make([]string, len(runtimeOptions))
	for i, opt := range runtimeOptions {
		names[i] = opt.Name
	}
	return names
}
