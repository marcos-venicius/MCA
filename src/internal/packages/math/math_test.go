package math

import (
	"math"
	"strings"
	"testing"

	"mca/internal/packages/pkgtest"
)

// m prefixes src with the package import, so a test reads the way a program
// using the package does.
func m(src string) string { return "const m = import('math')\n" + src }

func TestMathFunctions(t *testing.T) {
	pkgtest.CheckFloat(t, m("m.PI()"), math.Pi)
	pkgtest.CheckFloat(t, m("m.E()"), math.E)

	pkgtest.CheckInt(t, m("m.abs(-1)"), 1)
	pkgtest.CheckFloat(t, m("m.abs(-15.5)"), 15.5)
	pkgtest.CheckInt(t, m("m.abs(true)"), 1) // bools count as 0/1, staying integral
	pkgtest.CheckInt(t, m("m.abs(m.abs(-1) - 2)"), 1)

	pkgtest.CheckInt(t, m("m.sin(0)"), 0) // whole results collapse back to int
	pkgtest.CheckInt(t, m("m.cos(0)"), 1)
	pkgtest.CheckInt(t, m("m.tan(0)"), 0)
	pkgtest.CheckInt(t, m("m.deg(m.asin(1))"), 90)
	pkgtest.CheckInt(t, m("m.deg(m.acos(0))"), 90)
	pkgtest.CheckFloat(t, m("m.rad(180)"), math.Pi)
	pkgtest.CheckInt(t, m("m.deg(3.14159265358979323846)"), 180)

	pkgtest.CheckInt(t, m("m.sqrt(25)"), 5)
	pkgtest.CheckInt(t, m("m.log(1)"), 0)
	pkgtest.CheckInt(t, m("m.log10(1000)"), 3)
	pkgtest.CheckFloat(t, m("m.exp(1)"), math.E)
	pkgtest.CheckInt(t, m("m.floor(4.8)"), 4)
	pkgtest.CheckInt(t, m("m.ceil(4.2)"), 5)
	pkgtest.CheckInt(t, m("m.round(4.5)"), 5)
	pkgtest.CheckInt(t, m("m.round(4.4)"), 4)

	// composes with the builtins that stayed behind
	pkgtest.CheckFloat(t, m("max(m.abs(-12), 8) * m.sin(m.rad(30)) + (16 / 2)"), 14)
}

func TestWrongArgTypes(t *testing.T) {
	pkgtest.ExpectError(t, m("m.sqrt('25')"), "unexpected data type")
	pkgtest.ExpectError(t, m("m.abs([1])"), "unexpected data type")
	pkgtest.ExpectError(t, m("m.floor({})"), "unexpected data type")
	pkgtest.ExpectError(t, m("m.sin(?)"), "unexpected data type")
}

func TestArity(t *testing.T) {
	pkgtest.ExpectError(t, m("m.sqrt()"), "")
	pkgtest.ExpectError(t, m("m.sqrt(1, 2)"), "")
	pkgtest.ExpectError(t, m("m.PI(1)"), "")
}

// A package function is an ordinary value: storable, passable to the
// higher-order builtins, arity-checked when called through a value.
func TestFunctionsAreFirstClass(t *testing.T) {
	pkgtest.CheckInt(t, m("f = m.abs; f(-3)"), 3)
	pkgtest.CheckInt(t, m("sum(map([-1, -2, 3], m.abs))"), 6)
	pkgtest.ExpectError(t, m("f = m.abs; f(1, 2)"), "")
}

func TestHelp(t *testing.T) {
	for _, src := range []string{"help('math')", "help('math.sqrt')", "help(import('math').sqrt)"} {
		_, out, err := pkgtest.Run(t, src)
		if err != nil {
			t.Fatalf("%q: unexpected runtime error: %v", src, err)
		}

		if !strings.Contains(out, "math.sqrt(x: int|float|bool) -> int|float") {
			t.Errorf("%q: expected the signature of math.sqrt, got:\n%s", src, out)
		}
	}
}
