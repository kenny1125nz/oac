package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Executor struct {
}

func New() *Executor {
	return &Executor{}
}

func (e *Executor) Run(ctx context.Context, cmdStr string) (ExecResult, error) {
	shell := "bash"
	flag := "-c"

	cmd := exec.CommandContext(ctx, shell, flag, cmdStr)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	err := cmd.Run()

	result := ExecResult{
		Stdout: strings.TrimSuffix(stdoutBuf.String(), "\n"),
		Stderr: strings.TrimSuffix(stderrBuf.String(), "\n"),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
			result.Stderr = err.Error()
		}
	}

	return result, nil
}

func Quote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "$", "\\$")
	s = strings.ReplaceAll(s, "`", "\\`")
	return fmt.Sprintf("\"%s\"", s)
}
