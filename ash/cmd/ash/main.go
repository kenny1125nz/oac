package main

import (
	"fmt"
	"io"
	"os"

	"github.com/agentic-coding/ash/eval"
	"github.com/agentic-coding/ash/lexer"
	"github.com/agentic-coding/ash/parser"
)

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	var src string

	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", os.Args[1], err)
			return 1
		}
		src = string(data)
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
			return 1
		}
		src = string(data)
	}

	tokens, err := lexer.Tokenize(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tokenize error: %v\n", err)
		return 1
	}

	script, err := parser.Parse(tokens)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		return 1
	}

	e := eval.New()
	exitCode := e.EvalScriptWithExit(script)
	return exitCode
}
