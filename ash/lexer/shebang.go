package lexer

import (
	"fmt"
	"regexp"
	"strings"
)

type ShebangDecl struct {
	Engine  string
	Version string
	Model   string
}

type CompactConfig struct {
	Mode     string
	Window   string
	Strategy string
	Raw      map[string]string
}

var shebangRe = regexp.MustCompile(`^#!\s*(\w[\w.-]*)\s*:\s*(\S+?)(?:\s*:\s*(\S+?))?\s*$`)
var compactRe = regexp.MustCompile(`^#!compact\s+(.+)$`)

func ParseShebang(line string) (*ShebangDecl, error) {
	m := shebangRe.FindStringSubmatch(line)
	if m == nil {
		return nil, fmt.Errorf("invalid shebang: %q", line)
	}
	d := &ShebangDecl{
		Engine:  m[1],
		Version: m[2],
	}
	if len(m) > 3 {
		d.Model = strings.TrimSpace(m[3])
	}
	return d, nil
}

func ParseCompactLine(line string) *CompactConfig {
	m := compactRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	c := &CompactConfig{Raw: make(map[string]string)}
	pairs := strings.Fields(m[1])
	for _, p := range pairs {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		c.Raw[k] = v
		switch k {
		case "mode":
			c.Mode = v
		case "window":
			c.Window = v
		case "strategy":
			c.Strategy = v
		}
	}
	return c
}
