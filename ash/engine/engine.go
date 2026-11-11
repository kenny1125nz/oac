package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Options struct {
	Model    string
	Subagent string
	Dir      string
}

type Engine interface {
	Name() string
	Run(ctx context.Context, prompt string, opts *Options) *Result
}

func Get(name string) (Engine, bool) {
	e, ok := registry[name]
	return e, ok
}

var registry = map[string]Engine{}

func register(e Engine) {
	registry[e.Name()] = e
}

func shellOut(ctx context.Context, args []string, dir string) *Result {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Run()

	r := &Result{
		Stdout: strings.TrimSuffix(stdoutBuf.String(), "\n"),
		Stderr: strings.TrimSuffix(stderrBuf.String(), "\n"),
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			r.ExitCode = exitErr.ExitCode()
		} else {
			r.ExitCode = -1
			if r.Stderr == "" {
				r.Stderr = err.Error()
			}
		}
	}
	return r
}

type configEngine struct {
	name   string
	config AgentConfig
}

func (e *configEngine) Name() string { return e.name }

func (e *configEngine) Run(ctx context.Context, prompt string, opts *Options) *Result {
	if opts == nil {
		opts = &Options{}
	}

	args := []string{e.config.Cmd}
	args = append(args, e.config.Args...)

	if flag, ok := e.config.Flags["subagent"]; ok && opts.Subagent != "" && opts.Subagent != "default" {
		args = append(args, flag, opts.Subagent)
	}

	if flag, ok := e.config.Flags["model"]; ok && opts.Model != "" {
		args = append(args, flag, opts.Model)
	}

	if e.config.Prompt == "positional" {
		args = append(args, prompt)
	} else if e.config.Prompt != "" {
		args = append(args, e.config.Prompt, prompt)
	}

	return shellOut(ctx, args, opts.Dir)
}

func InitFromConfig(cfg *AgentsConfig) error {
	for name, agentCfg := range cfg.Agents {
		if agentCfg.Cmd == "" {
			return fmt.Errorf("agent %q has no cmd", name)
		}
		register(&configEngine{name: name, config: agentCfg})
	}
	return nil
}

func init() {
	cfg, err := LoadConfig()
	if err == nil {
		InitFromConfig(cfg)
	}
}
