package keys

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Ctrl-C", "Ctrl+C"},
		{"Ctrl-T", "Ctrl+T"},
		{"Rune[j]", "Rune[j]"},
		{"Enter", "Enter"},
		{"Escape", "Escape"},
		{"Ctrl-Shift-A", "Ctrl+Shift-A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
