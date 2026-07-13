package io

import (
	"os"
	"strings"
	"testing"

	"mca/internal/interp"
	"mca/internal/lexer"
	"mca/internal/packages/pkgtest"
	"mca/internal/parser"
)

func TestReadEntireFile(t *testing.T) {
	// Run() lexes with no filename, so this '.'-path passes through
	// ResolvePath unchanged and resolves against the working directory
	// (this package's directory, under `go test`).
	pkgtest.CheckString(t,
		"import('io').read_entire_file('../../../../test/file.txt')",
		"Hello World\n")
}

// A '.'-prefixed path resolves relative to the calling file's own directory,
// exactly like import() -- not the process working directory.
func TestPathResolvesRelativeToCallingFile(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/data.txt", []byte("from the script's dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	src := "import('io').read_entire_file('./data.txt')"
	if err := os.WriteFile(dir+"/main.mca", []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	l := lexer.New(dir+"/main.mca", src)
	toks := l.Tokenize()
	if len(l.Errors) > 0 {
		t.Fatalf("lex errors: %v", l.Errors)
	}

	prog := parser.Parse(dir+"/main.mca", toks)
	if len(prog.Errors) > 0 {
		t.Fatalf("parse errors: %v", prog.Errors)
	}

	in := interp.New()
	got, err := in.Run(prog.Stmts)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}

	sv, ok := got.(interp.StringValue)
	if !ok || string(sv) != "from the script's dir" {
		t.Fatalf("read_entire_file result = %+v, want the file's contents", got)
	}
}

func TestErrors(t *testing.T) {
	pkgtest.ExpectError(t, "import('io').read_entire_file('no_such_file.txt')", "could not read file")
	pkgtest.ExpectError(t, "import('io').read_entire_file(123)", "unexpected data type")
	pkgtest.ExpectError(t, "import('io').read_entire_file()", "")
}

func TestHelp(t *testing.T) {
	_, out, err := pkgtest.Run(t, "help('io.read_entire_file')")
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}

	if !strings.Contains(out, "io.read_entire_file(path: string) -> string") {
		t.Errorf("expected the signature of io.read_entire_file, got:\n%s", out)
	}
}
