package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	prompt := ""
	model := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--model" && i+1 < len(args) {
			model = args[i+1]
			i++
		} else if prompt == "" {
			prompt = args[i]
		} else {
			prompt += " " + args[i]
		}
	}

	if prompt == "" {
		prompt = "(empty prompt)"
	}

	fmt.Fprintf(os.Stdout, "Mock response for: %s\n", prompt)
	if model != "" {
		fmt.Fprintf(os.Stdout, "using model: %s\n", model)
	}
	fmt.Fprintf(os.Stderr, "mock-agent stderr output\n")
	os.Exit(0)
}
