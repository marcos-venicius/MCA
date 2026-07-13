// Package pkgtest is the shared test harness for native packages' tests.
//
// A package's tests live with the package rather than in interp's, and drive
// it through real MCA source: importing a package from interp's own tests
// would be an import cycle (the package -> interp), and the package's tests
// are the only place its registering init() is guaranteed to have run anyway.
// This harness is the lex/parse/run boilerplate those tests would otherwise
// each copy; it depends only on interp, so any package test may import it.
package pkgtest

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"mca/internal/interp"
	"mca/internal/lexer"
	"mca/internal/parser"
)

// Run lexes, parses, and runs src in a fresh interpreter. Returns the value
// of the last statement, everything the program printed, and any runtime
// error. Lex and parse errors fail the test immediately -- the tests here are
// about runtime behavior, never about syntax.
func Run(t *testing.T, src string) (interp.Value, string, error) {
	t.Helper()

	l := lexer.New("", src)
	toks := l.Tokenize()
	if len(l.Errors) > 0 {
		t.Fatalf("%q: lex errors: %v", src, l.Errors)
	}

	prog := parser.Parse("", toks)
	if len(prog.Errors) > 0 {
		t.Fatalf("%q: parse errors: %v", src, prog.Errors)
	}

	var out bytes.Buffer

	in := interp.New()
	in.Out = &out
	in.Err = &out

	v, err := in.Run(prog.Stmts)

	return v, out.String(), err
}

func value(t *testing.T, src string) interp.Value {
	t.Helper()

	v, _, err := Run(t, src)
	if err != nil {
		t.Fatalf("%q: unexpected runtime error: %v", src, err)
	}
	return v
}

func CheckInt(t *testing.T, src string, want int64) {
	t.Helper()

	v := value(t, src)
	iv, ok := v.(interp.IntValue)
	if !ok {
		t.Fatalf("%q: expected an int but got a '%s'", src, v.Kind())
	}
	if int64(iv) != want {
		t.Errorf("%q: got %d, want %d", src, int64(iv), want)
	}
}

func CheckFloat(t *testing.T, src string, want float64) {
	t.Helper()

	v := value(t, src)
	fv, ok := v.(interp.FloatValue)
	if !ok {
		t.Fatalf("%q: expected a float but got a '%s'", src, v.Kind())
	}
	got := float64(fv)
	if !(got == want || (math.IsNaN(got) && math.IsNaN(want))) {
		t.Errorf("%q: got %v, want %v", src, got, want)
	}
}

func CheckBool(t *testing.T, src string, want bool) {
	t.Helper()

	v := value(t, src)
	bv, ok := v.(interp.BoolValue)
	if !ok {
		t.Fatalf("%q: expected a bool but got a '%s'", src, v.Kind())
	}
	if bool(bv) != want {
		t.Errorf("%q: got %v, want %v", src, bool(bv), want)
	}
}

func CheckString(t *testing.T, src, want string) {
	t.Helper()

	v := value(t, src)
	sv, ok := v.(interp.StringValue)
	if !ok {
		t.Fatalf("%q: expected a string but got a '%s'", src, v.Kind())
	}
	if string(sv) != want {
		t.Errorf("%q: got %q, want %q", src, string(sv), want)
	}
}

func CheckUnit(t *testing.T, src string) {
	t.Helper()

	v := value(t, src)
	if v.Kind() != interp.KUnit {
		t.Errorf("%q: expected unit but got a '%s'", src, v.Kind())
	}
}

// ExpectError asserts src raises a runtime error; contains, when non-empty,
// must appear in its message.
func ExpectError(t *testing.T, src, contains string) {
	t.Helper()

	_, _, err := Run(t, src)
	if err == nil {
		t.Errorf("%q: expected a runtime error, got none", src)
		return
	}
	if contains != "" && !strings.Contains(err.Error(), contains) {
		t.Errorf("%q: got error %q, want it to contain %q", src, err.Error(), contains)
	}
}
