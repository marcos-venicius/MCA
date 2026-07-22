package main

import (
	"fmt"
	"os"
	"strings"

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
	fmt.Fprintf(w, "    --help-packages [name]\n")
	fmt.Fprintf(w, "                        print library documentation and exit: no name\n")
	fmt.Fprintf(w, "                        gives the general overview, a name documents one\n")
	fmt.Fprintf(w, "                        builtin, package, or member (e.g. 'math.sqrt')\n")
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "version: %s", version)
	fmt.Fprintf(w, "\n\n")
}

// helpPackages renders the standard-library documentation (the same content as
// the in-language help() builtin) straight to stdout, then reports the exit
// code. An empty name prints the general overview; a non-empty one documents a
// single builtin, package, or qualified member.
func helpPackages(name string) int {
	in := interp.New()
	if err := in.Help(name); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}

func run() int {
	programName := os.Args[0]
	rest := os.Args[1:]

	var inputFileName string
	var progArgs []string

	for i := range len(rest) {
		arg := rest[i]

		if arg == "-h" {
			usage(os.Stdout, programName)
			return 0
		}

		// --help-packages is a terminal action: it documents the standard
		// library and exits, never running a program. The name it documents may
		// be attached with '=' (--help-packages=math) or passed as the next
		// token (--help-packages math); with neither, it prints the overview.
		if flagVal, isFlag := strings.CutPrefix(arg, "--help-packages"); isFlag && (flagVal == "" || flagVal[0] == '=') {
			name := strings.TrimPrefix(flagVal, "=")
			if name == "" && i+1 < len(rest) && !strings.HasPrefix(rest[i+1], "-") {
				name = rest[i+1]
			}
			return helpPackages(name)
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
