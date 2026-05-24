package toolsutil_test

import (
	"strings"
	"testing"

	"dev-forge-mcp/internal/dpf"
	"dev-forge-mcp/internal/tools/toolsutil"
)

func TestErrResult(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		wantSub string
	}{
		{"empty message", "", `"error":""`},
		{"normal message", "something went wrong", `"error":"something went wrong"`},
		{"message with quotes", `say "hi"`, `"error":"say \"hi\""`},
		{"message with newline", "line1\nline2", `"error":"line1\nline2"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toolsutil.ErrResult(tt.msg)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("ErrResult(%q) = %q, want substring %q", tt.msg, got, tt.wantSub)
			}
		})
	}
}

func TestResultJSON(t *testing.T) {
	t.Run("simple struct", func(t *testing.T) {
		type kv struct {
			K string `json:"k"`
			V int    `json:"v"`
		}
		got := toolsutil.ResultJSON(kv{"hello", 42})
		if got != `{"k":"hello","v":42}` {
			t.Errorf("unexpected: %s", got)
		}
	})

	t.Run("nested struct", func(t *testing.T) {
		type inner struct {
			X int `json:"x"`
		}
		type outer struct {
			A inner `json:"a"`
		}
		got := toolsutil.ResultJSON(outer{inner{7}})
		if got != `{"a":{"x":7}}` {
			t.Errorf("unexpected: %s", got)
		}
	})

	t.Run("nil", func(t *testing.T) {
		got := toolsutil.ResultJSON(nil)
		if got != "null" {
			t.Errorf("unexpected: %s", got)
		}
	})

	t.Run("slice", func(t *testing.T) {
		got := toolsutil.ResultJSON([]int{1, 2, 3})
		if got != `[1,2,3]` {
			t.Errorf("unexpected: %s", got)
		}
	})

	t.Run("unmarshalable returns error envelope", func(t *testing.T) {
		got := toolsutil.ResultJSON(make(chan int))
		if !strings.Contains(got, `"error"`) {
			t.Errorf("expected error envelope, got: %s", got)
		}
		if !strings.Contains(got, "marshal failed") {
			t.Errorf("expected 'marshal failed' in error, got: %s", got)
		}
	})
}

func TestRequireDPF(t *testing.T) {
	t.Run("nil client returns error envelope and false", func(t *testing.T) {
		msg, ok := toolsutil.RequireDPF(nil)
		if ok {
			t.Fatal("expected ok=false for nil client")
		}
		if !strings.Contains(msg, `"error"`) {
			t.Errorf("expected error envelope, got: %s", msg)
		}
		if !strings.Contains(msg, "dpf binary not available") {
			t.Errorf("expected dpf message in envelope, got: %s", msg)
		}
	})

	t.Run("non-nil client returns empty string and true", func(t *testing.T) {
		client := &dpf.StreamClient{}
		msg, ok := toolsutil.RequireDPF(client)
		if !ok {
			t.Fatal("expected ok=true for non-nil client")
		}
		if msg != "" {
			t.Errorf("expected empty msg, got: %s", msg)
		}
	})
}
