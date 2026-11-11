package compact

import (
	"fmt"
	"strconv"
	"strings"
)

type Mode string

const (
	ModeAuto Mode = "auto"
	ModeOn   Mode = "on"
	ModeOff  Mode = "off"
)

type Strategy string

const (
	StrategyTruncate  Strategy = "truncate"
	StrategySummarize Strategy = "summarize"
	StrategyWindow    Strategy = "window"
	StrategyDrop      Strategy = "drop"
)

type Directive struct {
	Strategy Strategy
	Window   int
	Keep     string
	Pattern  string
}

func (d *Directive) String() string {
	switch d.Strategy {
	case StrategyTruncate:
		return fmt.Sprintf("truncate %d", d.Window)
	case StrategySummarize:
		return "summarize"
	case StrategyWindow:
		return fmt.Sprintf("window %d keep %s", d.Window, d.Keep)
	case StrategyDrop:
		return fmt.Sprintf("drop %s", d.Pattern)
	default:
		return "unknown"
	}
}

type Config struct {
	Mode     Mode
	Window   int
	Strategy Strategy
	Header   string
}

func NewConfig() *Config {
	return &Config{
		Mode:     ModeAuto,
		Window:   32000,
		Strategy: StrategyTruncate,
	}
}

func FromHeader(raw map[string]string) *Config {
	c := NewConfig()
	if raw == nil {
		return c
	}
	if m, ok := raw["mode"]; ok {
		c.Mode = Mode(m)
	}
	if w, ok := raw["window"]; ok {
		if n, err := strconv.Atoi(w); err == nil {
			c.Window = n
		}
	}
	if s, ok := raw["strategy"]; ok {
		c.Strategy = Strategy(s)
	}
	return c
}

func ParseDirective(s string) *Directive {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return nil
	}
	d := &Directive{}
	switch parts[0] {
	case "truncate":
		d.Strategy = StrategyTruncate
		if len(parts) > 1 {
			d.Window, _ = strconv.Atoi(parts[1])
		}
		if d.Window <= 0 {
			d.Window = 32000
		}
	case "summarize":
		d.Strategy = StrategySummarize
	case "window":
		d.Strategy = StrategyWindow
		if len(parts) > 1 {
			d.Window, _ = strconv.Atoi(parts[1])
		}
		if d.Window <= 0 {
			d.Window = 32000
		}
		for i, p := range parts {
			if p == "keep" && i+1 < len(parts) {
				d.Keep = strings.Join(parts[i+1:], " ")
				break
			}
		}
	case "drop":
		d.Strategy = StrategyDrop
		if len(parts) > 1 {
			d.Pattern = strings.Join(parts[1:], " ")
		}
	default:
		return nil
	}
	return d
}

func (c *Config) Active() bool {
	return c.Mode == ModeOn || (c.Mode == ModeAuto && c.Strategy != "")
}

func (c *Config) ApplyDirective(d *Directive) string {
	msg := fmt.Sprintf("[compact] %s", d.String())
	return msg
}
