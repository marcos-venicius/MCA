package main

import (
	"fmt"
	"os"

	"mca/internal/interp"
	"mca/internal/lexer"
	"mca/internal/parser"

	// Registers the native packages import() can resolve ('crypt', ...).
	_ "mca/internal/packages"
)

const version string = "0.1.0-go"

func usage(w *os.File, programName string) {
	fmt.Fprintf(w, "USAGE: %s <file> [argv]\n\n", programName)
	fmt.Fprintf(w, "    -h                  show this help\n")
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "version: %s", version)
	fmt.Fprintf(w, "\n\n")
}

func main() {
	os.Exit(run())
}

func run() int {
	programName := os.Args[0]
	rest := os.Args[1:]

	var inputFileName string
	var progArgs []string

	for _, arg := range rest {
		if arg == "-h" {
			usage(os.Stdout, programName)
			return 0
		}

		if inputFileName == "" {
			inputFileName = arg
		}

		progArgs = append(progArgs, arg)
	}

	if inputFileName == "" {
		usage(os.Stderr, programName)
		fmt.Fprintln(os.Stderr, "error: missing input file")
		return 1
	}

	content, err := os.ReadFile(inputFileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not open file %s due to: %s\n", inputFileName, err)
		return 1
	}

	l := lexer.New(inputFileName, string(content))
	tokens := l.Tokenize()

	if len(l.Errors) > 0 {
		for _, e := range l.Errors {
			fmt.Fprintln(os.Stderr, e.Error())
		}
		return 1
	}

	prog := parser.Parse(inputFileName, tokens)

	if len(prog.Errors) > 0 {
		for _, e := range prog.Errors {
			fmt.Fprintln(os.Stderr, e.Error())
		}
		fmt.Fprintf(os.Stderr, "parsing failed with \033[1;31m%d\033[0m errors\n", len(prog.Errors))
		return 1
	}

	in := interp.New()
	in.Args = progArgs

	if _, err := in.Run(prog.Stmts); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	return 0
}
