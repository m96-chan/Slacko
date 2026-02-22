package keys

import "strings"

// Normalize converts tcell key names to the config format.
// tcell outputs "Ctrl-C" (hyphen) for bare Ctrl keys but config uses "Ctrl+C" (plus).
func Normalize(name string) string {
	return strings.ReplaceAll(name, "Ctrl-", "Ctrl+")
}
