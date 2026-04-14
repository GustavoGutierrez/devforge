package main

import (
	"testing"

	"dev-forge-mcp/internal/version"
)

func TestVersionRequested(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: nil, want: false},
		{name: "version flag", args: []string{"--version"}, want: true},
		{name: "other arg", args: []string{"help"}, want: false},
		{name: "version among args", args: []string{"help", "--version"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionRequested(tt.args); got != tt.want {
				t.Fatalf("versionRequested(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestVersionOutput(t *testing.T) {
	want := "devforge v" + version.Current + "\n"

	if got := versionOutput(); got != want {
		t.Fatalf("versionOutput() = %q, want %q", got, want)
	}
}
