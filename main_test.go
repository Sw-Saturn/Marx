package main

import "testing"

func TestMatchCommand(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		command string
		want    bool
	}{
		{"exact match", "/approve", "/approve", true},
		{"with trailing newline", "/approve\n", "/approve", true},
		{"with surrounding text", "LGTM\n/approve\nthanks", "/approve", true},
		{"no match", "looks good", "/approve", false},
		{"partial match", "/approve-all", "/approve", false},
		{"custom command", "/lgtm", "/lgtm", true},
		{"whitespace around", "  /approve  ", "/approve", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchCommand(tt.body, tt.command)
			if got != tt.want {
				t.Errorf("matchCommand(%q, %q) = %v, want %v", tt.body, tt.command, got, tt.want)
			}
		})
	}
}
