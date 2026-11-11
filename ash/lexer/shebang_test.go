package lexer

import "testing"

func TestParseShebang(t *testing.T) {
	tests := []struct {
		line    string
		engine  string
		version string
		model   string
	}{
		{"#!opencode:1.2.0", "opencode", "1.2.0", ""},
		{"#!opencode:1.2.0:sonnet", "opencode", "1.2.0", "sonnet"},
		{"#!claude-code:2.0.0:claude-sonnet-4", "claude-code", "2.0.0", "claude-sonnet-4"},
		{"#!aider:v0.45.0", "aider", "v0.45.0", ""},
	}

	for _, tt := range tests {
		decl, err := ParseShebang(tt.line)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tt.line, err)
			continue
		}
		if decl.Engine != tt.engine {
			t.Errorf("%q: engine = %q, want %q", tt.line, decl.Engine, tt.engine)
		}
		if decl.Version != tt.version {
			t.Errorf("%q: version = %q, want %q", tt.line, decl.Version, tt.version)
		}
		if decl.Model != tt.model {
			t.Errorf("%q: model = %q, want %q", tt.line, decl.Model, tt.model)
		}
	}
}

func TestParseShebangInvalid(t *testing.T) {
	invalid := []string{
		"# not shebang",
		"#!",
		"#!foo",
		"//!opencode:1.0",
		"",
	}
	for _, line := range invalid {
		_, err := ParseShebang(line)
		if err == nil {
			t.Errorf("%q: expected error", line)
		}
	}
}

func TestParseCompactConfig(t *testing.T) {
	tests := []struct {
		line   string
		mode   string
		window string
		strat  string
	}{
		{"#!compact mode=on window=64000 strategy=truncate", "on", "64000", "truncate"},
		{"#!compact mode=auto", "auto", "", ""},
	}

	for _, tt := range tests {
		cc := ParseCompactLine(tt.line)
		if cc == nil {
			t.Errorf("%q: expected non-nil config", tt.line)
			continue
		}
		if cc.Mode != tt.mode {
			t.Errorf("%q: mode = %q, want %q", tt.line, cc.Mode, tt.mode)
		}
		if cc.Window != tt.window {
			t.Errorf("%q: window = %q, want %q", tt.line, cc.Window, tt.window)
		}
		if cc.Strategy != tt.strat {
			t.Errorf("%q: strategy = %q, want %q", tt.line, cc.Strategy, tt.strat)
		}
	}
}

func TestParseCompactConfigEmpty(t *testing.T) {
	cc := ParseCompactLine("#!compact")
	if cc != nil {
		t.Logf("empty compact line returned config: %+v", cc)
	}
}
