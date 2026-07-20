package crypt

import (
	"bytes"
	"strings"
	"testing"

	"mca/internal/interp"
	"mca/internal/lexer"
	"mca/internal/parser"
)

// A package's tests live with the package rather than in interp's, and drive
// it through real MCA source: importing crypt from interp_test.go would be an
// import cycle (crypt -> interp), and this is the only place the init() that
// registers 'crypt' is guaranteed to have run anyway.
func run(t *testing.T, src string) (interp.Value, string, error) {
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

func checkString(t *testing.T, src, want string) {
	t.Helper()

	got, _, err := run(t, src)
	if err != nil {
		t.Fatalf("%q: unexpected runtime error: %v", src, err)
	}

	if got.Kind() != interp.KString {
		t.Fatalf("%q: expected a string but got a '%s'", src, got.Kind())
	}

	if s := interp.AsString(got); s != want {
		t.Errorf("%q: got %q, want %q", src, s, want)
	}
}

func TestMD5(t *testing.T) {
	// RFC 1321 test vectors.
	checkString(t, "import('crypt').md5('')", "d41d8cd98f00b204e9800998ecf8427e")
	checkString(t, "import('crypt').md5('a')", "0cc175b9c0f1b6a831c399e269772661")
	checkString(t, "import('crypt').md5('abc')", "900150983cd24fb0d6963f7d28e17f72")
	checkString(t, "import('crypt').md5('message digest')", "f96b697d7cb7938d525a2f31aaf161d0")
}

// The map import() returns is an ordinary MCA value, so a package function is
// storable and passable like any other builtin -- not a name that only works
// in call position.
func TestMD5IsFirstClass(t *testing.T) {
	checkString(t, "const crypt = import('crypt')\nh = crypt.md5\nh('a')", "0cc175b9c0f1b6a831c399e269772661")
	checkString(t, "map(['a'], import('crypt').md5)[0]", "0cc175b9c0f1b6a831c399e269772661")
}

// Each import() builds its own map, so mutating one importer's view of the
// package cannot reach another's.
// A package's members are constants: the map import() returns is frozen, so
// assigning over a function (or any other member) is a runtime error rather
// than a silent clobber.
func TestPackageIsImmutable(t *testing.T) {
	_, _, err := run(t, "a = import('crypt')\na.md5 = 'clobbered'")
	if err == nil {
		t.Fatal("expected a runtime error assigning to a package member, got none")
	}

	if !strings.Contains(err.Error(), "cannot modify a frozen object") {
		t.Errorf("got error %q, want it to mention the object is frozen", err.Error())
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"import('crypt').md5(1)", "expected a 'string' but got a 'int'"},
		{"import('nope')", "there is no package named 'nope'"},
		// A path is always a file, never a package: 'crypt' the package must
		// not answer for './crypt.mca' the file, nor shadow it.
		{"import('./crypt.mca')", "could not open module"},
		{"help('crypt.sha256')", "package 'crypt' has no member named 'sha256'"},
	}

	for _, tt := range tests {
		_, _, err := run(t, tt.src)
		if err == nil {
			t.Errorf("%q: expected a runtime error, got none", tt.src)
			continue
		}

		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%q: got error %q, want it to contain %q", tt.src, err.Error(), tt.want)
		}
	}
}

func TestHelp(t *testing.T) {
	// help('crypt') documents the package, help('crypt.md5') one function,
	// and help(crypt.md5) resolves the same doc through the value itself.
	for _, src := range []string{"help('crypt')", "help('crypt.md5')", "help(import('crypt').md5)"} {
		_, out, err := run(t, src)
		if err != nil {
			t.Fatalf("%q: unexpected runtime error: %v", src, err)
		}

		if !strings.Contains(out, "crypt.md5(s: string) -> string") {
			t.Errorf("%q: expected the signature of crypt.md5, got:\n%s", src, out)
		}
	}

	// The overview lists the package, so a program can find it without
	// knowing it exists beforehand.
	_, out, err := run(t, "help()")
	if err != nil {
		t.Fatalf("help(): unexpected runtime error: %v", err)
	}

	if !strings.Contains(out, "crypt") {
		t.Errorf("help(): expected the overview to list the crypt package, got:\n%s", out)
	}
}
