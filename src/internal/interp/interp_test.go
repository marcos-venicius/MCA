package interp

import (
	"bytes"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"mca/internal/lexer"
	"mca/internal/parser"
)

func newTestInterp(args ...string) *Interp {
	in := New()
	in.Out = io.Discard
	in.Err = io.Discard
	in.Args = args
	return in
}

type expected struct {
	kind Kind
	i    int64
	f    float64
	b    bool
	s    string
}

func tUnit() expected           { return expected{kind: KUnit} }
func tInt(v int64) expected     { return expected{kind: KInt, i: v} }
func tFloat(v float64) expected { return expected{kind: KFloat, f: v} }
func tBool(v bool) expected     { return expected{kind: KBool, b: v} }
func tString(v string) expected { return expected{kind: KString, s: v} }

func check(t *testing.T, src string, want expected) {
	t.Helper()
	checkArgs(t, src, want)
}

func checkArgs(t *testing.T, src string, want expected, args ...string) {
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

	in := newTestInterp(args...)
	got, err := in.Run(prog.Stmts)
	if err != nil {
		t.Fatalf("%q: unexpected runtime error: %v", src, err)
	}

	if got.Kind() != want.kind {
		t.Fatalf("%q: kind = %v, want %v (full value %+v)", src, got.Kind(), want.kind, got)
	}

	switch want.kind {
	case KUnit:
		// nothing else to check
	case KInt:
		if gotI := intOf(got); gotI != want.i {
			t.Fatalf("%q: = %d, want %d", src, gotI, want.i)
		}
	case KFloat:
		gotF := floatOf(got)
		if !(gotF == want.f || (math.IsNaN(gotF) && math.IsNaN(want.f))) {
			t.Fatalf("%q: = %v, want %v", src, gotF, want.f)
		}
	case KBool:
		if gotB := boolOf(got); gotB != want.b {
			t.Fatalf("%q: = %v, want %v", src, gotB, want.b)
		}
	case KString:
		if gotS := stringOf(got); gotS != want.s {
			t.Fatalf("%q: = %q, want %q", src, gotS, want.s)
		}
	}
}

// checkPrints runs src and asserts on what it wrote to stdout. It replaces
// the format(...)-based result serialization these tests used before format
// moved to the 'string' package: native packages register from init()s that
// interp's own tests never link (importing one here would be an import
// cycle), so anything moved out is unreachable from this file -- println's
// value printing serializes arrays just as well.
func checkPrints(t *testing.T, src, want string) {
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

	var buf bytes.Buffer
	in := New()
	in.Out = &buf
	in.Err = io.Discard

	if _, err := in.Run(prog.Stmts); err != nil {
		t.Fatalf("%q: unexpected runtime error: %v", src, err)
	}

	if got := buf.String(); got != want {
		t.Fatalf("%q: printed %q, want %q", src, got, want)
	}
}

func expectRuntimeError(t *testing.T, src string) {
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

	in := newTestInterp()
	_, err := in.Run(prog.Stmts)
	if err == nil {
		t.Fatalf("%q: expected a runtime error, got none", src)
	}
}

// expectParseError asserts src is rejected before it ever runs -- for things
// the grammar itself forbids, like `const` without an initializer.
func expectParseError(t *testing.T, src string) {
	t.Helper()

	l := lexer.New("", src)
	toks := l.Tokenize()

	prog := parser.Parse("", toks)
	if len(prog.Errors) == 0 {
		t.Fatalf("%q: expected a parse error, got none", src)
	}
}

func TestTopLevel(t *testing.T) {
	check(t, "", tUnit())
	check(t, ";", tUnit())
}

func TestBasicArithmetic(t *testing.T) {
	check(t, "1 + 2", tInt(3))
	check(t, "10 - 5", tInt(5))
	check(t, "3 * 4", tInt(12))
	check(t, "20 / 4", tInt(5))
	check(t, "10 % 3", tInt(1))
}

func TestOperatorPrecedence(t *testing.T) {
	check(t, "1 + 2 * 3", tInt(7))
	check(t, "(1 + 2) * 3", tInt(9))
	check(t, "10 - 4 / 2", tInt(8))
	check(t, "(10 - 4) / 2", tInt(3))
}

func TestExponentiationRightAssociative(t *testing.T) {
	check(t, "2 ^ 3", tInt(8))
	check(t, "2 ^ 3 ^ 2", tInt(512))
	check(t, "(2 ^ 3) ^ 2", tInt(64))
}

func TestUnaryOperators(t *testing.T) {
	check(t, "-5", tInt(-5))
	check(t, "-(-5)", tInt(5))
	check(t, "4!", tInt(24))
	check(t, "-4!", tInt(-24))
	check(t, "(-4)!", tFloat(math.NaN()))

	// prefix operators chain right-to-left, no parens needed
	check(t, "--1", tInt(1))          // -(-1)
	check(t, "---5", tInt(-5))        // -(-(-5))
	check(t, "- -5", tInt(5))         // whitespace between doesn't matter
	check(t, "!!false", tBool(false)) // !(!false)
	check(t, "!!true", tBool(true))
	check(t, "!!!false", tBool(true))
	check(t, "~~5", tInt(5))      // ~(~5)
	check(t, "!-1", tBool(false)) // !(-1): -1 is truthy
	check(t, "-~0", tInt(1))      // -(~0) = -(-1)
	check(t, "--4!", tInt(24))    // factorial still binds tighter: -(-(4!))
}

func TestCombinations(t *testing.T) {
	check(t, "2 * 3! + 4 ^ 2 / -2", tInt(4))
	check(t, "-((((5.0! + -20) / 10) ^ 2) % 11) * 10 * -1 - 10", tFloat(0))
}

func TestBinaryOperatorsEqualityAndRelational(t *testing.T) {
	check(t, "5 == 5", tBool(true))
	check(t, "10 == 5", tBool(false))
	check(t, "0 == 0", tBool(true))
	check(t, "10 != 5", tBool(true))
	check(t, "5 != 5", tBool(false))
	check(t, "0 != 0", tBool(false))
	check(t, "10 > 5", tBool(true))
	check(t, "5 > 10", tBool(false))
	check(t, "5 > 5", tBool(false))
	check(t, "5 < 10", tBool(true))
	check(t, "10 < 5", tBool(false))
	check(t, "5 < 5", tBool(false))
	check(t, "10 >= 5", tBool(true))
	check(t, "5 >= 5", tBool(true))
	check(t, "5 >= 10", tBool(false))
	check(t, "5 <= 10", tBool(true))
	check(t, "5 <= 5", tBool(true))
	check(t, "10 <= 5", tBool(false))
	check(t, "1 + 2 == 3", tBool(true))
	check(t, "10 - 5 > 2 * 2", tBool(true))
	check(t, "5 < 3 + 4", tBool(true))
	check(t, "10 == 5 * 2 != 0", tBool(true))
	check(t, "0 == 1 < 2", tBool(false))
	check(t, "'Hello' == 'hello'", tBool(false))
	check(t, "'Hello' == 'Hello'", tBool(true))
	check(t, "'Hello World' == 'Hello'", tBool(false))
	check(t, "'Hello World' != 'Hello'", tBool(true))
	check(t, "'Hello' != 'Hello'", tBool(false))
}

func TestShiftOperators(t *testing.T) {
	check(t, "1 << 3", tInt(8))
	check(t, "16 >> 2", tInt(4))
	check(t, "1 << 0", tInt(1))
	check(t, "5 >> 0", tInt(5))
	check(t, "0 << 5", tInt(0))
	check(t, "7 >> 1", tInt(3))    // low bits truncate
	check(t, "-16 >> 2", tInt(-4)) // arithmetic shift: sign preserved
	check(t, "-1 >> 63", tInt(-1)) // the sign bit fills all the way down
	check(t, "1 << 64", tInt(0))   // over-wide shifts drain to 0, they don't wrap
	check(t, "16 >> 64", tInt(0))
	check(t, "a = 2; a << a", tInt(8))

	// left-associative chain
	check(t, "1 << 4 >> 2", tInt(4))

	// precedence: '+' binds tighter, '<'/'==' bind looser
	check(t, "1 + 1 << 2", tInt(8))
	check(t, "1 << 2 + 1", tInt(8))
	check(t, "16 >> 1 > 7", tBool(true))
	check(t, "1 << 2 == 4", tBool(true))
}

func TestShiftWrongOperands(t *testing.T) {
	// int-only: no float/bool coercion, unlike '+' and friends
	expectRuntimeError(t, "1.5 << 1")
	expectRuntimeError(t, "1 << 1.5")
	expectRuntimeError(t, "true << 1")
	expectRuntimeError(t, "1 >> false")
	expectRuntimeError(t, "'a' << 1")
	expectRuntimeError(t, "1 << 'a'")
	expectRuntimeError(t, "[1] << 1")
	expectRuntimeError(t, "? << 1")

	// a negative shift count has no defined meaning
	expectRuntimeError(t, "1 << -1")
	expectRuntimeError(t, "1 >> -1")
}

func TestBitwiseOperators(t *testing.T) {
	check(t, "5 ~ 3", tInt(6)) // binary '~' is xor
	check(t, "5 ~ 5", tInt(0))
	check(t, "5 & 3", tInt(1))
	check(t, "5 | 3", tInt(7))
	check(t, "-1 & 7", tInt(7)) // two's complement: -1 is all ones
	check(t, "-2 | 1", tInt(-1))

	// prefix '~' is bitwise not
	check(t, "~0", tInt(-1))
	check(t, "~5", tInt(-6))
	check(t, "~(-1)", tInt(0))
	check(t, "a = 5; ~a", tInt(-6))
	check(t, "5 ~ ~3", tInt(-7)) // xor with a bitwise-not operand

	// precedence, Lua-style: & over ~ over |, all under the shifts and
	// over the comparisons
	check(t, "1 | 2 ~ 2 & 3", tInt(1))  // 1 | (2 ~ (2 & 3))
	check(t, "1 ~ 1 << 3", tInt(9))     // 1 ~ (1 << 3)
	check(t, "1 ~ 3 == 2", tBool(true)) // (1 ~ 3) == 2
	check(t, "2 | 1 == 3", tBool(true)) // (2 | 1) == 3
	check(t, "~5 + 1", tInt(-5))        // unary binds tighter: (~5) + 1
}

func TestBitwiseWrongOperands(t *testing.T) {
	// int-only, like the shifts
	expectRuntimeError(t, "1.5 ~ 1")
	expectRuntimeError(t, "1 ~ 1.5")
	expectRuntimeError(t, "1 & true")
	expectRuntimeError(t, "false | 1")
	expectRuntimeError(t, "'a' ~ 1")
	expectRuntimeError(t, "1 & 'a'")
	expectRuntimeError(t, "[1] | 1")
	expectRuntimeError(t, "? ~ 1")

	expectRuntimeError(t, "~1.5")
	expectRuntimeError(t, "~true")
	expectRuntimeError(t, "~'a'")
	expectRuntimeError(t, "~[1]")
	expectRuntimeError(t, "~?")
}

func TestSum(t *testing.T) {
	check(t, "sum([1, 2, 3])", tInt(6))
	check(t, "sum([-1, -2, 3])", tInt(0))
	check(t, "sum([5])", tInt(5)) // single element
	check(t, "sum([])", tInt(0))  // empty array -- sums to int 0
	check(t, "sum([1.5, 2.5])", tFloat(4.0))
	check(t, "sum([1, 2.5])", tFloat(3.5)) // any float present -> promotes to float
	check(t, "sum([1, 2.5, 3])", tFloat(6.5))

	check(t, "type(sum([1, 2, 3]))", tString("int"))
	check(t, "type(sum([1, 2.0, 3]))", tString("float"))

	// composes with other array builtins
	check(t, "sum(reverse([1, 2, 3]))", tInt(6))
	check(t, "sum(sort([3, 1, 2], \\(x, y) -> x - y))", tInt(6))
	check(t, "sum(map([1, 2, 3], \\(x) -> x * 2))", tInt(12))
	check(t, "sum(filter([1, 2, 3, 4], \\(x) -> x > 2))", tInt(7))

	// source array untouched
	check(t, "a = [1, 2, 3]; sum(a); len(a)", tInt(3))
}

func TestSumWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "sum(123)") // must be an array
	expectRuntimeError(t, "sum('not an array')")
	expectRuntimeError(t, "sum({})")
	expectRuntimeError(t, "sum([1, 2, 'a'])") // every element must be int|float
	expectRuntimeError(t, "sum([1, true])")   // bools aren't accepted, unlike abs/max/min
	expectRuntimeError(t, "sum([1, [2]])")
}

func TestSumArity(t *testing.T) {
	expectRuntimeError(t, "sum()")
	expectRuntimeError(t, "sum([1, 2], [3])")
}

func TestCallBuiltinFunctions(t *testing.T) {
	// sin, sqrt, abs and friends moved to the 'math' package, ord/
	// format to 'string', srand/rand to 'random', read_entire_file to 'io' --
	// each is tested in its own package (internal/packages/...), since their
	// registering init()s never run in this test binary.
	check(t, "max(min(3, 8), 2) * min(4, 5)", tInt(12))
	check(t, "max(10.5, 20.0)", tFloat(20.0))
	check(t, "min(10.5, 20.0)", tFloat(10.5))
	check(t, "type(4.4)", tString("float"))
	check(t, "type(4)", tString("int"))
	check(t, "len('Hello World')", tInt(11))
	checkArgs(t, "argc()", tInt(0))
	checkArgs(t, "argc()", tInt(1), "fakename.mca")
	checkArgs(t, "argv(0)", tString("fakename.mca"), "fakename.mca")
	checkArgs(t, "argv(1)", tString("fakearg"), "fakename.mca", "fakearg")
	check(t, "is_typeof(1, 'int')", tBool(true))
	check(t, "is_typeof(1.3, 'int')", tBool(false))
	check(t, "is_typeof(1, 'float')", tBool(false))
	check(t, "is_typeof(1.3, 'float')", tBool(true))
	check(t, "is_typeof(1, 'string')", tBool(false))
	check(t, "is_typeof('1.3', 'string')", tBool(true))
	check(t, "is_typeof(1, 'bool')", tBool(false))
	check(t, "is_typeof(false, 'bool')", tBool(true))
	check(t, "is_typeof(1, 'unit')", tBool(false))
	check(t, "is_typeof(?, 'unit')", tBool(true))
	check(t, "is_typeof([], 'array')", tBool(true))
	check(t, "is_typeof({}, 'map')", tBool(true))
	check(t, "is_typeof([], 'map')", tBool(false))
	check(t, "'Hello, World'[7]", tString("W"))
}

func TestPrintingReturnsLastArgument(t *testing.T) {
	check(t, "print()", tUnit())
	check(t, "print(3.14)", tFloat(3.14))
	check(t, "print(3.14, 2.71, 10)", tInt(10))
	check(t, "println()", tUnit())
	check(t, "println(3.14)", tFloat(3.14))
	check(t, "println(3.14, 2.71, 10)", tInt(10))
}

func TestGlobalVariables(t *testing.T) {
	check(t, "x = 10", tInt(10))
	check(t, "y = x = 10", tInt(10))
	check(t, "y = x = 10;y", tInt(10))
	check(t, "x = 10; y = -5.5; z = -(x * y); println(x, y, z, x + y + z)", tFloat(59.5))
}

func TestAssignment(t *testing.T) {
	check(t, "y = x = 10; x + y", tInt(20))
	check(t, "i = 0; while i < 10 { i += 1 }", tInt(10))
	check(t, "i = 10; while i > 10 { i -= 1 }", tUnit())
	check(t, "i = 10; i += 2", tInt(12))
	check(t, "i = 10; i += 2; i", tInt(12))
	check(t, "i = 10; i -= 2", tInt(8))
	check(t, "i = 10; i -= 2; i", tInt(8))
	check(t, "m = {}; m['name'] = 'Fred'; m['age'] = 32; m['name']", tString("Fred"))
	check(t, "m = {}; m['name'] = 'Fred'; m['age'] = 32; m['age']", tInt(32))
}

func TestWhileLoops(t *testing.T) {
	check(t, "n = 10; while n < 20 { n = n + 1 }", tInt(20))
	check(t, "a = 0; b = 1; n = 0; while n < 15 { n = n + 1; t = a; a = b; b = t + b; a }", tInt(610)) // fib
	check(t, "while false {}", tUnit())
	check(t, "while false;", tUnit())
	check(t, "n = 0; while n < 10  n += 1", tInt(10))
	check(t, "i = 0; n = 0; while n < 10  n += 1  i += 1", tInt(1)) // bodyless while accepts a single expression as body
}

func TestForLoops(t *testing.T) {
	check(t, "for i : 10;", tUnit())
	check(t, "for i : 10 { i + 1 }", tInt(10))
	check(t, "n = 10; for i : n { i + 1 }", tInt(10))
	check(t, "x = 0; for i : [10, -1, -1] { x += i }", tInt(55))
	check(t, "m = { 'x': 10, 'y': 15, 'z': 2 }; r = 0; for k, v : m { if (k == 'x' or k == 'y') r += v }; r", tInt(25))
	check(t, "r = 0; text = 'Hello, World'; for i, v : text { if v == 'o' { r += i } }; r", tInt(12))
	check(t, "r = 1; a = [1, 2, 3, 4, 5] for _, v : a { r = r * v }", tInt(120))
}

func TestBreak(t *testing.T) {
	check(t, "r = while 1 { n = 10; break 11.3; println(0); }; r", tFloat(11.3))
	check(t, "r = while 1 { n = 10; break; println(10); }; r", tUnit())
	check(t, "r = while 1 { n = 10; break 10 * 10 - 1; println(10); }; r", tInt(99))
}

func TestForLoopBreakIsFixed(t *testing.T) {
	check(t, "r = 0; for i : 10 { if i == 3 { break; }; r = i }; r", tInt(2))
	check(t, "r = for i : 10 { if i == 3 { break 99 } }; r", tInt(99))
}

func TestContinue(t *testing.T) {
	// for-range: continue skips the body for i == 2, so the sum omits it.
	check(t, "s = 0; for i : 5 { if i == 2 continue; s = s + i }; s", tInt(8))
	// while: continue jumps back to the condition without ending the loop.
	check(t, "s = 0; n = 0; while n < 5 { n = n + 1; if n == 3 continue; s = s + n }; s", tInt(12))
	// for-of over an array skips one element.
	check(t, "s = 0; for _, v : [1, 2, 3, 4] { if v == 3 continue; s = s + v }; s", tInt(7))
	// continue and break coexist: break still ends the loop.
	check(t, "s = 0; for i : 10 { if i == 1 continue; if i == 4 break; s = s + i }; s", tInt(5))
}

func TestContinueOutsideLoopIsRuntimeError(t *testing.T) {
	expectRuntimeError(t, "continue")
}

func TestIfs(t *testing.T) {
	check(t, "x = 10; if x == 10 { x = 11.3 }", tFloat(11.3))
	check(t, "if 0 == 0;", tUnit())
	check(t, "x = if 10 != 10.1 { 1337 }", tInt(1337))
	check(t, "x = if 10 != 10.1 {}", tUnit())
	check(t, "if false 0 elif false 1 else true 2", tInt(2))
	check(t, "if false { 0 } elif false { 1 } else true 2", tInt(2))
	check(t, "if false { 0 } elif false { 1 } else { 2; 3; 4; }", tInt(4))
	check(t, "n = 4; a = if n % 2 == 0 'Ok' else 'Fail'; println(a)", tString("Ok"))
}

func TestElifs(t *testing.T) {
	check(t, "if 0 == 1; elif 0 == 0;", tUnit())
	check(t, "if 10 == 10.1 { 1337 } elif 20 == 21 { 1 } elif 20 == 20 { 56 } elif 1 == 1 { } else { 42 }", tInt(56))
	check(t, "if 10 == 10.1 { 1337 } elif 20 == 21 { 1 } elif 20 == 22 { 56 } elif 1 == 1 { 33 } else { 42 }", tInt(33))
	check(t, "if 10 == 10 { 1337 } elif 20 == 21 { 1 } elif 20 == 22 { 56 } elif 1 == 1 { 33 } else { 42 }", tInt(1337))
	check(t, "if 10 == 11 { 1337 } elif 20 == 21 { 1 } elif 20 == 22 { 56 } elif 2 == 1 { 33 } else { 42 }", tInt(42))
	check(t, "if false 0 elif true { ;;;; } else true", tUnit())
}

func TestElses(t *testing.T) {
	check(t, "if 0 == 1; elif 0 == 1; else;", tUnit())
	check(t, "if 10 == 10.1 { 1337 } else { 42 }", tInt(42))
	check(t, "if 10 == 10.1 { 1337 } else { }", tUnit())
	check(t, "if 10 == 10.1 { 1337 } else { ;; }", tUnit())
}

func TestLogicalOperators(t *testing.T) {
	check(t, "if 10 == 10 and 20 == 20 { 20 } else { }", tInt(20))
	check(t, "if 10 == 11 and 20 == 20 { 20 } else { 10 }", tInt(10))
	check(t, "if 10 == 11 or 20 == 20 { 20 } else { 10 }", tInt(20))
	check(t, "if 10 == 11 or 20 == 21 { 20 } else { 10 }", tInt(10))
	check(t, "if 10 == 10 or n == x { 20 } else { 0 }", tInt(20))  // right side is lazily evaluated
	check(t, "if 10 == 11 and n == x { 0 } else { 20 }", tInt(20)) // right side is lazily evaluated
}

func TestLogicalOperatorsTruthinessAcrossAllKinds(t *testing.T) {
	// and/or use Truthy(), which is defined for every value kind -- unlike
	// the generic binary-op path, arrays/maps/fns/unit are all legal
	// operands here, not just int/float/bool.
	check(t, "0 and true", tBool(false))
	check(t, "1 and true", tBool(true))
	check(t, "0.0 and true", tBool(false))
	check(t, "1.5 and true", tBool(true))
	check(t, "false and true", tBool(false))
	check(t, "'' and true", tBool(false))
	check(t, "'x' and true", tBool(true))
	check(t, "[] and true", tBool(false))
	check(t, "[1] and true", tBool(true))
	check(t, "{} and true", tBool(false))
	check(t, "{'a': 1} and true", tBool(true))
	check(t, "? and true", tBool(false))               // unit is always falsy
	check(t, "f = \\() -> 1; f and true", tBool(true)) // fn values are always truthy

	check(t, "0 or false", tBool(false))
	check(t, "1 or false", tBool(true))
	check(t, "'' or false", tBool(false))
	check(t, "'x' or false", tBool(true))
	check(t, "[] or false", tBool(false))
	check(t, "[1] or false", tBool(true))
	check(t, "{} or false", tBool(false))
	check(t, "{'a': 1} or false", tBool(true))
	check(t, "? or false", tBool(false))
	check(t, "f = \\() -> 1; f or false", tBool(true))
}

func TestBinaryOpsRejectArrayMapFnOperands(t *testing.T) {
	// The generic binary-op path (everything but and/or) type-checks both
	// operands and only accepts int/float/bool/unit/string for arithmetic
	// and relational operators -- arrays, maps, and fns are always rejected
	// there, regardless of what the other operand is.
	expectRuntimeError(t, "[1] + [2]")
	expectRuntimeError(t, "[1] < [2]")
	expectRuntimeError(t, "{} + {}")
	expectRuntimeError(t, "f = \\() -> 1; f + 1")
}

func TestEqualityShortCircuitsOnKindMismatchBeforeTypeCheck(t *testing.T) {
	// == and != compare Kind() before the arithmetic/relational type check
	// runs, so comparing an array/map/fn against a value of a *different*
	// kind is legal and just reports "not equal" -- the type check never
	// sees the array/map/fn operand.
	check(t, "[1] == 'x'", tBool(false))
	check(t, "[1] != 'x'", tBool(true))
	check(t, "{} == 1", tBool(false))
	check(t, "{} != 1", tBool(true))
	check(t, "f = \\() -> 1; f == 1", tBool(false))
	check(t, "f = \\() -> 1; f != 1", tBool(true))
	check(t, "[1] == {}", tBool(false))
	check(t, "? == [1]", tBool(false))
	check(t, "? != [1]", tBool(true))
}

func TestEqualityCoercesNumericKinds(t *testing.T) {
	// int/float/bool are one coercible numeric family everywhere else in
	// evalBinary (e.g. '1 < 1.5' and 'true < 2' both coerce) -- == and !=
	// must agree with that instead of comparing Kind() first.
	check(t, "1 == 1.0", tBool(true))
	check(t, "1.5 == 1.5", tBool(true))
	check(t, "1 == 1.5", tBool(false))
	check(t, "true == 1", tBool(true))
	check(t, "false == 0", tBool(true))
	check(t, "true == 1.0", tBool(true))
	check(t, "false == 2", tBool(false))
	check(t, "1 != 1.0", tBool(false))
	check(t, "true != 0", tBool(true))
}

func TestBinaryOpTypeChecksLeftBeforeEvaluatingRight(t *testing.T) {
	// A bad left-operand type must be reported without evaluating the right
	// operand at all, same as before equality gained its own eager-eval
	// path -- otherwise a runtime error on the right (e.g. div by zero)
	// would mask the left operand's type error.
	src := "[1] + (1 % 0)"

	l := lexer.New("", src)
	toks := l.Tokenize()
	prog := parser.Parse("", toks)

	_, err := newTestInterp().Run(prog.Stmts)
	if err == nil {
		t.Fatalf("%q: expected a runtime error, got none", src)
	}
	if !strings.Contains(err.Error(), "array") {
		t.Fatalf("%q: expected the left operand's type error (mentioning 'array'), got: %v", src, err)
	}
}

func TestEqualitySupportsSameKindArrayMapFn(t *testing.T) {
	// Two operands of the *same* kind (array/array, map/map, fn/fn) compare
	// structurally for arrays/maps (element-by-element, recursively) and by
	// identity for fns.
	check(t, "[1] == [1]", tBool(true))
	check(t, "[1] != [1]", tBool(false))
	check(t, "{} == {}", tBool(true))
	check(t, "{} != {}", tBool(false))
	check(t, "f = \\() -> 1; f == f", tBool(true))
	check(t, "f = \\() -> 1; f != f", tBool(false))
}

func TestNotOperator(t *testing.T) {
	check(t, "if !(10 == 11 or 20 == 21) { 20 } else { 10 }", tInt(20))
	check(t, "!(1 == 0)", tBool(true))
	check(t, "!1.4", tBool(false))
}

func TestBooleans(t *testing.T) {
	check(t, "true", tBool(true))
	check(t, "false", tBool(false))
	check(t, "!true", tBool(false))
	check(t, "!false", tBool(true))
}

func TestTypeCasting(t *testing.T) {
	check(t, "as_int(10.5)", tInt(10))
	check(t, "as_int(true)", tInt(1))
	check(t, "as_int('-103956')", tInt(-103956))
	check(t, "as_int('103956')", tInt(103956))
	check(t, "as_int('103956'[2])", tInt(3))
	check(t, "as_float(10)", tFloat(10.0))
	check(t, "as_float(false)", tFloat(0.0))
	check(t, "as_float('-23.2356')", tFloat(-23.2356))
	check(t, "as_float('23.56')", tFloat(23.56))
	check(t, "as_bool(10)", tBool(true))
	check(t, "as_bool(0)", tBool(false))
	check(t, "as_bool(false)", tBool(false))
	check(t, "as_bool(true)", tBool(true))
	check(t, "as_string(10234)", tString("10234"))
	check(t, "as_string(true)", tString("true"))
	check(t, "as_string(false)", tString("false"))
	check(t, "as_string(-120)", tString("-120"))
	check(t, "as_string(120.234)", tString("120.234"))
	check(t, "as_string(-120.234)", tString("-120.234"))
	check(t, "as_string(1.32)", tString("1.32")) // exact value, no trailing zeros
	check(t, "as_string(1.0)", tString("1.0"))   // a whole float keeps its ".0"
	check(t, "as_string(0.5)", tString("0.5"))
}

// is_typeof takes the kind to test for as a string, so a name that isn't one
// of type()'s eight kinds is a runtime error rather than a silent false.
func TestIsTypeof(t *testing.T) {
	check(t, `is_typeof(\(x) -> x, 'fn')`, tBool(true))
	check(t, "is_typeof(1, type(1))", tBool(true)) // type()'s output round-trips
	check(t, "is_typeof(1.5, type(1.5))", tBool(true))
	check(t, "is_typeof([], type([]))", tBool(true))

	expectRuntimeError(t, "is_typeof(1, 'number')") // no such kind
	expectRuntimeError(t, "is_typeof(1, 'Int')")    // kind names are lowercase
	expectRuntimeError(t, "is_typeof(1, '')")
	expectRuntimeError(t, "is_typeof(1, 2)") // the kind must be a string
	expectRuntimeError(t, "is_typeof(1)")    // both arguments are required
	expectRuntimeError(t, "is_typeof(1, 'int', 'float')")
}

// Floats print and stringify as the shortest decimal that round-trips, never
// in scientific notation -- an exact 1.32 is "1.32", not "1.320000". print,
// println, and as_string all share this via FormatFloat.
func TestFloatFormatting(t *testing.T) {
	// print path matches as_string
	checkPrints(t, "println(1.32)", "1.32\n")
	checkPrints(t, "println(1.0)", "1.0\n") // whole float keeps ".0"
	checkPrints(t, "println(67.56)", "67.56\n")
	checkPrints(t, "print(3.14, 2.71)", "3.142.71") // print doesn't pad either

	// full precision is preserved (no truncation to 6 digits)
	check(t, "as_string(3.141592653589793)", tString("3.141592653589793"))

	// no scientific notation for large/small magnitudes
	check(t, "as_string(1000000.0)", tString("1000000.0"))
	check(t, "as_string(0.0001)", tString("0.0001"))

	// NaN and infinities render as themselves, with no spurious ".0"
	check(t, "as_string((-4)!)", tString("NaN"))
	check(t, "as_string(1.0 / 0.0)", tString("+Inf"))
	check(t, "as_string(-1.0 / 0.0)", tString("-Inf"))
}

func TestUnitType(t *testing.T) {
	check(t, "?", tUnit())
	check(t, "a = ?", tUnit())
}

func TestStringTypeLiterals(t *testing.T) {
	check(t, "'Hello, World'", tString("Hello, World"))
	check(t, "a = 'Hello, World'", tString("Hello, World"))
	check(t, "println('Hello, World')", tString("Hello, World"))
	check(t, `print('Hello, World\n')`, tString("Hello, World\n"))
}

func TestFunctions(t *testing.T) {
	check(t, `f = \(a, b) -> a + b; f(10, 20)`, tInt(30))
	check(t, `f = \() -> 100; f()`, tInt(100))
	check(t, `f = \(x) -> { x * 2 }; f(10)`, tInt(20))
	check(t, `f = \(x) -> { a = 10; x + a }; f(5)`, tInt(15))
	check(t, `f = \(a, cb) -> cb(a); f(10, \(x) -> x * 2)`, tInt(20))
	check(t, `make_adder = \(x) -> \(y) -> x + y; add_5 = make_adder(5); add_5(10)`, tInt(15))
	check(t, `x = 10; f = \(x) -> x * 2; f(20) + x`, tInt(50))
	check(t, `x = 10; f = \() -> x; x = 20; f()`, tInt(20))
	check(t, `f = \(x) -> x; if true { y = 100; f(y) }`, tInt(100))
}

func TestFunctionClosuresDoNotAlias(t *testing.T) {
	// calling the same lambda literal multiple
	// times must capture an independent environment each time, not share
	// one mutable closure_env across every value produced from that literal.
	check(t, `make_adder = \(x) -> \(y) -> x + y; add_5 = make_adder(5); add_10 = make_adder(10); add_5(1)`, tInt(6))
	check(t, `make_adder = \(x) -> \(y) -> x + y; add_5 = make_adder(5); add_10 = make_adder(10); add_10(1)`, tInt(11))
}

func TestCallOnAnyExpression(t *testing.T) {
	// '(' is a general postfix operator now, not something recognized only
	// after a bare identifier -- any expression that evaluates to a
	// function can be called immediately.
	check(t, `(\() -> 42)()`, tInt(42))                        // IIFE on a parenthesized fn literal
	check(t, `(\(x) -> x * 2)(21)`, tInt(42))                  // IIFE with an argument
	check(t, `a = [\(x) -> x + 1]; a[0](41)`, tInt(42))        // call on an array-index expression
	check(t, `m = {'f': \(x) -> x + 1}; m['f'](41)`, tInt(42)) // call on a map-index expression
	check(t, `f = \() -> \(x) -> x + 1; f()(41)`, tInt(42))    // chained call: call the result of a call
	check(t, `m = {'a': {'f': \() -> 42}}; m.a.f()`, tInt(42)) // call through a multi-level dot chain
}

func TestCalleeMustBeAFunction(t *testing.T) {
	expectRuntimeError(t, "x = 5; x()")
	expectRuntimeError(t, "a = [1, 2]; a[0]()")
	expectRuntimeError(t, "m = {}; m.missing()") // reading a missing field is unit; calling unit is a runtime error
	expectRuntimeError(t, "(1 + 1)()")
}

func TestBuiltinsCanBeShadowed(t *testing.T) {
	// Builtins live in a frame below the global scope, so an assignment to one
	// of their names binds a new variable in the assigning scope rather than
	// writing through to the builtin -- a program is free to use `len` or
	// `year` as a variable name.
	check(t, `len = \(x) -> 999; len('hi')`, tInt(999))
	check(t, `len = 3; len`, tInt(3))
	check(t, `const len = 3; len`, tInt(3))

	// The right-hand side still sees the builtin, which is what makes the
	// `year = year(0)` idiom in the examples work.
	check(t, `len = len('hi'); len`, tInt(2))

	// Shadowing is confined to the scope that did it: the builtin is untouched
	// everywhere else, including after the shadowing scope has been left.
	check(t, `f = \() -> { len = 1; len }; f(); len('hi')`, tInt(2))
	check(t, `f = \() -> { len = 1; len }; f()`, tInt(1))
	check(t, `for i : 3 { sort = 1 }; len(sort([2, 1], \(a, b) -> a - b))`, tInt(2))

	// A binding introduced by the scope itself (a parameter, a loop variable)
	// shadows a builtin too.
	check(t, `f = \(len) -> len + 1; f(10)`, tInt(11))
	check(t, `r = 0; for len : 4 { r += len }; r`, tInt(6))
}

func TestConst(t *testing.T) {
	check(t, `const x = 42; x`, tInt(42))
	check(t, `const x = 1 + 2; x * 2`, tInt(6))
	check(t, `const x = 5`, tInt(5)) // a declaration evaluates to its value
	check(t, `const f = \(x) -> x * 2; f(21)`, tInt(42))

	// const freezes the *name*, not the value it points at: the binding can
	// never be rewritten, but a mutable value reached through it still is.
	check(t, `const a = [1, 2]; a[0] = 9; a[0]`, tInt(9))
	check(t, `const a = [1, 2]; append(a, 3); len(a)`, tInt(3))
	check(t, `const m = {}; m.k = 7; m.k`, tInt(7))

	// An inner scope may shadow an outer constant with a binding of its own,
	// and doing so leaves the outer one untouched.
	check(t, `const x = 1; f = \(x) -> x * 10; f(5)`, tInt(50))
	check(t, `const x = 1; f = \(x) -> x * 10; f(5); x`, tInt(1))
	check(t, `const x = 1; if true { const x = 2 }; x`, tInt(1))

	// A const declaration in a loop body is fresh on every iteration, since
	// each iteration gets its own scope.
	check(t, `r = 0; for i : 3 { const c = i * 2; r += c }; r`, tInt(6))
	check(t, `i = 0; r = 0; while i < 3 { const c = 2; r += c; i += 1 }; r`, tInt(6))
}

func TestConstCannotBeReassigned(t *testing.T) {
	expectRuntimeError(t, `const x = 1; x = 2`)
	expectRuntimeError(t, `const x = 1; x += 1`)
	expectRuntimeError(t, `const x = 1; x -= 1`)

	// Reassignment is refused however deep the write is: Assign climbs to the
	// constant and stops there rather than quietly creating a local.
	expectRuntimeError(t, `const x = 1; f = \() -> { x = 2 }; f()`)
	expectRuntimeError(t, `const x = 1; if true { x = 2 }`)
	expectRuntimeError(t, `const x = 1; for i : 3 { x = 2 }`)

	// Redeclaring a constant next to itself isn't a constant.
	expectRuntimeError(t, `const x = 1; const x = 2`)
	expectRuntimeError(t, `x = 1; const x = 2`)

	// The value survives the failed writes above (each program is its own
	// run, so this just re-checks the binding is intact after a caught error).
	check(t, `const x = 1; x`, tInt(1))
}

func TestConstParseErrors(t *testing.T) {
	expectParseError(t, `const x`)
	expectParseError(t, `const x += 1`)
	expectParseError(t, `const 5 = 1`)

	// `const` is a soft keyword, like if/while: it is only a declaration when
	// an identifier follows, so it stays usable as a plain variable name.
	check(t, `const = 7; const + 1`, tInt(8))
}

func TestBuiltinsAreFirstClassValues(t *testing.T) {
	check(t, `type(len)`, tString("fn"))
	check(t, `is_typeof(sort, 'fn')`, tBool(true))

	// Stored in a variable, then called through it.
	check(t, `f = len; f('hi')`, tInt(2))
	check(t, `m = {'f': len}; m.f('hello')`, tInt(5))
	check(t, `fns = [min, max]; fns[1](1, 2)`, tInt(2))

	// Passed to the higher-order builtins as a callback.
	check(t, `sum(map(['a', 'bb'], len))`, tInt(3))
	check(t, `len(filter(['', 'a', ''], as_bool))`, tInt(1))
	check(t, `map([1, 2], as_string)[1]`, tString("2"))

	// Passed to a user function, and returned from one.
	check(t, `apply = \(f, x) -> f(x); apply(len, 'abc')`, tInt(3))
	check(t, `pick = \() -> len; pick()('hi')`, tInt(2))

	// A variadic builtin accepts any argument count, so it is usable wherever
	// a callback of a fixed arity is expected.
	check(t, `len(map([1, 2], println))`, tInt(2))

	// Arity is still enforced when a builtin is called through a value.
	expectRuntimeError(t, `f = len; f()`)
	expectRuntimeError(t, `f = len; f('a', 'b')`)
	expectRuntimeError(t, `map([1, 2], sort)`)
}

func TestReturn(t *testing.T) {
	check(t, `f = \() -> { return 42; 100 }; f()`, tInt(42))
	check(t, `f = \(x) -> { if x == 1 { return 10 } return 20 }; f(1)`, tInt(10))
	check(t, `f = \(x) -> { if x == 1 { return 10 } return 20 }; f(2)`, tInt(20))
	check(t, `f = \(x) -> { while x < 10 { if x == 5 { return x } x += 1 } return 0 }; f(0)`, tInt(5))
	check(t, `f = \(x) -> { while x < 10 { if x == 5 { return x } x += 1 } return 0 }; f(6)`, tInt(0))
	check(t, `f = \() -> { return; 100 }; f()`, tUnit())
}

func TestBreakOutsideLoopIsRuntimeError(t *testing.T) {
	expectRuntimeError(t, "break;")
}

func TestReturnOutsideFunctionIsRuntimeError(t *testing.T) {
	expectRuntimeError(t, "return;")
}

func TestStringOpTypeCheckIsSymmetric(t *testing.T) {
	expectRuntimeError(t, "5 + 'hi'")
	expectRuntimeError(t, "'hi' + 5")
	expectRuntimeError(t, "true + 'hi'")
}

func TestCompoundAssignRejectsNonNumericRight(t *testing.T) {
	expectRuntimeError(t, "x = 1; x += 'hi'")
	expectRuntimeError(t, "x = 1; x += true")
}

func TestModuloByZeroIsCleanRuntimeError(t *testing.T) {
	expectRuntimeError(t, "5 % 0")
}

func TestHashmaps(t *testing.T) {
	check(t, "m = {}; m[1] = 'Hello, World'; m[1]", tString("Hello, World"))
	expectRuntimeError(t, "m = {}; m[1] = 'Hello, World'; m[2]")
	check(t, "m = {}; m[1] = 'Hello, World';", tString("Hello, World"))
	check(t, "m = { 'name': 'John Doe', 'age': 32, 'weight': 67.56, 'is_dead': false, 10: 'test' }; m['age']", tInt(32))
	check(t, "m = {'name': 'John Doe','age': 32,'weight': 67.56,'is_dead': false,10: 'test'}; len(m)", tInt(5))
	check(t, "m = {'name': 'John Doe','age': 32,'weight': 67.56,'is_dead': false,10: 'test'}; m['name']", tString("John Doe"))
	check(t, "m = {'name': 'John Doe','age': 32,'weight': 67.56,'is_dead': false,10: 'test'}; m['weight']", tFloat(67.56))
	check(t, "m = {'name': 'John Doe','age': 32,'weight': 67.56,'is_dead': false,10: 'test'}; m['is_dead']", tBool(false))
	check(t, "m = {'name': 'John Doe','age': 32,'weight': 67.56,'is_dead': false,10: 'test'}; m[10]", tString("test"))
	check(t, "m = {}; len(m);", tInt(0))

	// Key removal is delete(m, key) -- the same builtin as on arrays; map_del
	// and map_clear are gone (clearing is rebinding to {}).
	check(t, "m = {}; m['width'] = '3rem'; m['height'] = '3rem'; m['z-index'] = 999; delete(m, 'height'); len(m)", tInt(2))
	check(t, "m = {}; m['width'] = '3rem'; m['height'] = '3rem'; m['z-index'] = 999; delete(m, 'height'); contains(m, 'height')", tBool(false))
	check(t, "m = {}; m['width'] = '3rem'; m['height'] = '3rem'; delete(m, 'Height'); len(m)", tInt(2)) // case-sensitive; missing key is not an error
	check(t, "m = {'a': 1}; m = {}; len(m)", tInt(0))
}

func TestHashmapFloatKeys(t *testing.T) {
	// float keys, set via index-assign and via a map literal
	check(t, "m = {}; m[1.5] = 'x'; m[1.5]", tString("x"))
	check(t, "m = {1.5: 'x'}; m[1.5]", tString("x"))
	check(t, "m = {-2.5: 'x'}; m[-2.5]", tString("x")) // negative float key

	// updating an existing float key doesn't grow the map
	check(t, "m = {}; m[1.5] = 'a'; m[1.5] = 'b'; m[1.5]", tString("b"))
	check(t, "m = {}; m[1.5] = 'a'; m[1.5] = 'b'; len(m)", tInt(1))

	// int and float keys are distinct even when numerically equal: MapKey
	// carries the Kind, so 1 and 1.0 never collide.
	check(t, "m = {}; m[1] = 'i'; m[1.0] = 'f'; len(m)", tInt(2))
	check(t, "m = {}; m[1] = 'i'; m[1.0] = 'f'; m[1]", tString("i"))
	check(t, "m = {}; m[1] = 'i'; m[1.0] = 'f'; m[1.0]", tString("f"))

	// float keys coexist with string/int keys in the same map
	check(t, "m = {1.5: 'a', 2.5: 'b', 'k': 'c', 10: 'd'}; len(m)", tInt(4))
	check(t, "m = {1.5: 'a', 2.5: 'b', 'k': 'c', 10: 'd'}; m[2.5]", tString("b"))

	// keys() round-trips a float key back to a float value
	check(t, "m = {}; m[2.5] = 'x'; keys(m)[0]", tFloat(2.5))

	// looking up a missing float key is a clean runtime error, not a panic
	expectRuntimeError(t, "m = {}; m[1.5]")
	expectRuntimeError(t, "m = {1.5: 'a'}; m[2.5]")

	// contains() and delete() accept float keys
	check(t, "m = {}; m[1.5] = 'x'; contains(m, 1.5)", tBool(true))
	check(t, "m = {1.5: 'x'}; contains(m, 2.5)", tBool(false))
	check(t, "m = {}; m[1.5] = 'x'; delete(m, 1.5); contains(m, 1.5)", tBool(false))

	// for-of yields the float key itself (not an empty string)
	check(t, "m = {2.5: 'a'}; found = false; for k, v : m { if k == 2.5 { found = true } }; found", tBool(true))
	check(t, "m = {1.5: 10, 2.5: 20}; s = 0.0; for k, v : m { s += k }; s", tFloat(4.0))

	// printing renders the float key (not an empty string); single-key maps
	// keep the output deterministic since iteration order isn't guaranteed
	checkPrints(t, "println({1.5: 'a'})", "{1.5: 'a'}\n")
	checkPrints(t, "println({-2.5: 10})", "{-2.5: 10}\n")
}

func TestMapKeys(t *testing.T) {
	check(t, "m = {}; len(keys(m))", tInt(0)) // empty map

	check(t, "m = {}; m['only'] = 42; ks = keys(m); len(ks)", tInt(1))
	check(t, "m = {}; m['only'] = 42; keys(m)[0]", tString("only"))

	check(t, "m = {}; m[7] = 'seven'; keys(m)[0]", tInt(7))

	// map iteration order isn't guaranteed, so verify the key *set* by
	// building a membership map out of the result rather than indexing by
	// position.
	check(t,
		"m = {}; m['width'] = '3rem'; m['height'] = '3rem'; m['z-index'] = 999;"+
			"ks = keys(m); found = {}; for i, k : ks { found[k] = true };"+
			"len(ks) == 3 and found['width'] and found['height'] and found['z-index']",
		tBool(true))

	// mixed string/int keys
	check(t,
		"m = {'name': 'John Doe', 'age': 32, 'weight': 67.56, 'is_dead': false, 10: 'test'};"+
			"ks = keys(m); found = {}; for i, k : ks { found[k] = true };"+
			"len(ks) == 5 and found['name'] and found['age'] and found['weight'] and found['is_dead'] and found[10]",
		tBool(true))

	// the returned array is a fresh copy -- mutating it doesn't touch the map
	check(t, "m = {}; m['a'] = 1; ks = keys(m); append(ks, 'z'); len(m)", tInt(1))
}

func TestMapKeysWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "keys(123)")
	expectRuntimeError(t, "keys('a string')")
	expectRuntimeError(t, "keys([1, 2, 3])")
	expectRuntimeError(t, "keys(true)")
}

func TestMapKeysArity(t *testing.T) {
	expectRuntimeError(t, "keys()")
	expectRuntimeError(t, "m = {}; keys(m, 'extra')")
}

func TestMapValues(t *testing.T) {
	check(t, "m = {}; len(values(m))", tInt(0)) // empty map

	check(t, "m = {}; m['only'] = 42; vs = values(m); len(vs)", tInt(1))
	check(t, "m = {}; m['only'] = 42; values(m)[0]", tInt(42))

	// map iteration order isn't guaranteed, so verify the value *set* by
	// building a membership map out of the result rather than indexing by
	// position.
	check(t,
		"m = {}; m['a'] = 1; m['b'] = 2; m['c'] = 3;"+
			"vs = values(m); found = {}; for i, v : vs { found[v] = true };"+
			"len(vs) == 3 and found[1] and found[2] and found[3]",
		tBool(true))

	// mixed-type values -- just check the count, since floats/bools can't
	// be used as map keys to check membership the same way
	check(t,
		"m = {'name': 'John Doe', 'age': 32, 'weight': 67.56, 'is_dead': false, 10: 'test'};"+
			"len(values(m))",
		tInt(5))

	// the returned array is a fresh copy -- mutating it doesn't touch the map
	check(t, "m = {}; m['a'] = 1; vs = values(m); append(vs, 99); len(m)", tInt(1))
}

func TestMapValuesWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "values(123)")
	expectRuntimeError(t, "values('a string')")
	expectRuntimeError(t, "values([1, 2, 3])")
	expectRuntimeError(t, "values(true)")
}

func TestMapValuesArity(t *testing.T) {
	expectRuntimeError(t, "values()")
	expectRuntimeError(t, "m = {}; values(m, 'extra')")
}

func TestArrays(t *testing.T) {
	check(t, "a = []; len(a)", tInt(0))
	check(t, "a = [1, 2, 'three']; len(a)", tInt(3))
	check(t, "a = [1, 2, 'three']; a[0]", tInt(1))
	check(t, "a = [1, 2, 'three']; a[2]", tString("three"))
	check(t, "a = [1]; append(a, 2); len(a)", tInt(2))
	check(t, "a = [1]; append(a, 2); a[1]", tInt(2))
}

func TestArrayIndexOutOfBounds(t *testing.T) {
	expectRuntimeError(t, "a = [1, 2, 3]; a[3]")
	expectRuntimeError(t, "a = [1, 2, 3]; a[-1]")
}

// The range operator base[from:to] slices strings and arrays -- a half-open
// [from, to) window, replacing the old string.select builtin. from must land
// in [0, len-1], to in [0, len], and from <= to.
func TestRangeExpression(t *testing.T) {
	// strings
	check(t, "'Hello, World'[7:12]", tString("World"))
	check(t, "'heyhey'[0:6]", tString("heyhey")) // whole string
	check(t, "'heyhey'[2:3]", tString("y"))      // single char
	check(t, "'heyhey'[3:6]", tString("hey"))
	check(t, "s = 'Hello, World'; s[7:len(s)]", tString("World")) // computed 'to' up to length
	check(t, "'hey'[1:1]", tString(""))                           // from == to yields empty

	// arrays -- check contents via println to avoid immediate index chaining
	checkPrints(t, "println([1, 2, 3, 4, 5][2:4])", "[3, 4]\n")
	checkPrints(t, "a = [1, 2, 3, 4, 5]; println(a[0:len(a)])", "[1, 2, 3, 4, 5]\n") // whole array
	checkPrints(t, "a = [1, 2, 3]; println(a[1:1])", "[]\n")                         // empty slice
	check(t, "a = [1, 2, 3, 4, 5]; b = a[2:4]; len(b)", tInt(2))
	check(t, "a = [1, 2, 3, 4, 5]; b = a[2:4]; b[0]", tInt(3))
	check(t, "a = [1, 2, 3, 4, 5]; b = a[2:4]; b[1]", tInt(4))

	// a range is a postfix operator like [i]/.field/(...), so it chains: the
	// result can be immediately indexed, sliced again, dotted, or called.
	check(t, "a = [1, 2, 3, 4, 5]; a[2:4][0]", tInt(3))                      // range then index
	check(t, "'hello'[0:3][0]", tString("h"))                                // range then index on a string
	check(t, "a = [1, 2, 3, 4, 5]; a[1:5][0:2][1]", tInt(3))                 // range then range then index
	check(t, "m = [[1, 2, 3], [4, 5, 6], [7, 8, 9]]; m[0:3][0][1]", tInt(2)) // slice rows, then [row][col]
	check(t, "a = [{'k': 42}]; a[0:1][0].k", tInt(42))                       // range then index then dot
	check(t, "a = [\\(x) -> x + 1]; a[0:1][0](41)", tInt(42))                // range then index then call
	expectRuntimeError(t, "a = [1, 2, 3, 4, 5]; a[2:4][5]")                  // chained index is a real bounds check now

	// an array slice is a fresh top-level array -- mutating or appending to it
	// never writes through to the source, and vice versa
	check(t, "a = [1, 2, 3, 4, 5]; b = a[1:3]; b[0] = 99; a[1]", tInt(2))
	check(t, "a = [1, 2, 3, 4, 5]; b = a[1:3]; append(b, 99); a[3]", tInt(4)) // no tail clobber
	check(t, "a = [1, 2, 3, 4, 5]; b = a[1:3]; a[1] = 77; b[0]", tInt(2))
	// ...but the copy is shallow: nested containers are still shared, like
	// Python's slice -- writing through an element reaches the original.
	check(t, "m = [[1, 2, 3], [4, 5, 6]]; s = m[0:2]; s[0][0] = 99; m[0][0]", tInt(99))

	// out-of-range / inverted bounds are runtime errors, on both kinds
	expectRuntimeError(t, "'hey'[-1:2]") // from negative
	expectRuntimeError(t, "'hey'[0:4]")  // to past length
	expectRuntimeError(t, "'hey'[2:1]")  // from > to
	expectRuntimeError(t, "'hey'[3:3]")  // from == length (from must be < length)
	expectRuntimeError(t, "''[0:0]")     // empty base has no valid range
	expectRuntimeError(t, "a = [1, 2, 3]; a[-1:2]")
	expectRuntimeError(t, "a = [1, 2, 3]; a[0:4]")
	expectRuntimeError(t, "a = [1, 2, 3]; a[2:1]")

	// bounds must be ints, and the base must be a string or array
	expectRuntimeError(t, "a = [1, 2, 3]; a[1.5:2]")
	expectRuntimeError(t, "a = [1, 2, 3]; a[0:1.5]")
	expectRuntimeError(t, "5[0:1]")
	expectRuntimeError(t, "m = {}; m[0:1]")
}

func TestIndexesToKeys(t *testing.T) {
	check(t, "r = indexes_to_keys(['x', 'y', 'z'], {0: 'first', 2: 'third'}); len(r)", tInt(2))
	check(t, "r = indexes_to_keys(['x', 'y', 'z'], {0: 'first', 2: 'third'}); r['first']", tString("x"))
	check(t, "r = indexes_to_keys(['x', 'y', 'z'], {0: 'first', 2: 'third'}); r['third']", tString("z"))

	check(t, "len(indexes_to_keys(['x', 'y', 'z'], {}))", tInt(0)) // empty obj -> empty result
	check(t, "len(indexes_to_keys([], {}))", tInt(0))              // empty array, empty obj

	check(t, "r = indexes_to_keys([1, 2, 3], {1: 'mid'}); r['mid']", tInt(2))   // non-string array elements
	check(t, "r = indexes_to_keys(['x', 'y'], {0: 100}); r[100]", tString("x")) // int target key, not just string
	check(t, "r = indexes_to_keys(['x', 'y'], {0: 1.5}); r[1.5]", tString("x")) // float target key

	// obj's value doesn't have to be the same kind as the array's elements,
	// and picking every index just renames them all
	check(t, "r = indexes_to_keys(['a', 'b'], {0: 0, 1: 1}); r[0]", tString("a"))
	check(t, "r = indexes_to_keys(['a', 'b'], {0: 0, 1: 1}); r[1]", tString("b"))

	// source array is untouched, and a fresh map is returned each call
	check(t, "a = [1, 2, 3]; indexes_to_keys(a, {0: 'x'}); len(a)", tInt(3))
	check(t, "a = ['x', 'y']; r1 = indexes_to_keys(a, {0: 'k'}); r2 = indexes_to_keys(a, {0: 'k'}); delete(r1, 'k'); len(r2)", tInt(1))
}

func TestIndexesToKeysErrors(t *testing.T) {
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], {'not_int': 'a'})") // obj key must be an int
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], {5: 'a'})")         // index out of range (too high)
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], {-1: 'a'})")        // index out of range (negative)
	expectRuntimeError(t, "indexes_to_keys([], {0: 'a'})")                 // any index into an empty array

	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], {0: true})") // obj value must be a valid map key
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], {0: [1]})")
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], {0: {}})")

	// a float obj *key* is still rejected -- keys index into the array, so
	// they must be ints (the error formats the float cleanly, not a panic)
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], {1.5: 'a'})")
}

func TestIndexesToKeysWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "indexes_to_keys(123, {0: 'a'})") // first arg must be an array
	expectRuntimeError(t, "indexes_to_keys('not an array', {0: 'a'})")
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], 123)") // second arg must be a map
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], ['a'])")
}

func TestIndexesToKeysArity(t *testing.T) {
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'])")
	expectRuntimeError(t, "indexes_to_keys(['x', 'y'], {0: 'a'}, 'extra')")
}

func TestSort(t *testing.T) {
	check(t, `a = sort([3, 1, 2], \(x, y) -> x - y); len(a)`, tInt(3))
	check(t, `a = sort([3, 1, 2], \(x, y) -> x - y); a[0]`, tInt(1))
	check(t, `a = sort([3, 1, 2], \(x, y) -> x - y); a[1]`, tInt(2))
	check(t, `a = sort([3, 1, 2], \(x, y) -> x - y); a[2]`, tInt(3))

	check(t, `a = sort([3, 1, 2], \(x, y) -> y - x); a[0]`, tInt(3)) // descending, via a flipped comparator
	check(t, `a = sort([3, 1, 2], \(x, y) -> y - x); a[2]`, tInt(1))

	check(t, `len(sort([], \(x, y) -> x - y))`, tInt(0))         // empty array
	check(t, `a = sort([5], \(x, y) -> x - y); len(a)`, tInt(1)) // single element -- unchanged
	check(t, `a = sort([5], \(x, y) -> x - y); a[0]`, tInt(5))

	checkPrints(t, `println(sort([3, 1, 2, 1, 3], \(x, y) -> x - y))`, "[1, 1, 2, 3, 3]\n") // duplicates preserved

	// comparator must itself return an int -- int subtraction works
	// directly, but a float comparator has to say explicitly which way it
	// goes since a - b would come out a float
	checkPrints(t,
		`cmp = \(x, y) -> if (x < y) { -1 } elif (x > y) { 1 } else { 0 };`+
			`println(sort([1.5, 0.5, 2.5], cmp))`,
		"[0.5, 1.5, 2.5]\n")

	// composes with other array builtins
	checkPrints(t, `println(sort(reverse([1, 2, 3]), \(x, y) -> x - y))`, "[1, 2, 3]\n")

	// the source array is untouched, and the result is a fresh array
	check(t, `a = [3, 1, 2]; b = sort(a, \(x, y) -> x - y); a[0]`, tInt(3))
	check(t, `a = [3, 1, 2]; b = sort(a, \(x, y) -> x - y); append(b, 9); len(a)`, tInt(3))
}

func TestSortWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, `sort(123, \(x, y) -> x - y)`) // first arg must be an array
	expectRuntimeError(t, `sort('not an array', \(x, y) -> x - y)`)
	expectRuntimeError(t, `sort([1, 2, 3], 123)`)       // second arg must be a function
	expectRuntimeError(t, `sort([1, 2, 3], \(x) -> x)`) // comparator must take exactly two arguments
	expectRuntimeError(t, `sort([1, 2, 3], \(x, y, z) -> x)`)
	expectRuntimeError(t, `sort([1, 2, 3], \(x, y) -> x > y)`)  // comparator must return an int, not a bool
	expectRuntimeError(t, `sort([1, 2, 3], \(x, y) -> 'lt')`)   // ... nor a string
	expectRuntimeError(t, `sort([1.5, 0.5], \(x, y) -> x - y)`) // ... nor a float (x - y on floats)
}

func TestSortArity(t *testing.T) {
	expectRuntimeError(t, `sort([1, 2, 3])`)
	expectRuntimeError(t, `sort([1, 2, 3], \(x, y) -> x - y, 'extra')`)
}

func TestDelete(t *testing.T) {
	// single-index form
	check(t, "a = [1, 2, 3, 4, 5]; delete(a, 2); len(a)", tInt(4))
	checkPrints(t, "a = [1, 2, 3, 4, 5]; delete(a, 2); println(a)", "[1, 2, 4, 5]\n")
	check(t, "a = [1]; delete(a, 0); len(a)", tInt(0)) // delete the only element

	// range form -- half-open [start, end), matching the range operator s[from:to]
	check(t, "a = [1, 2, 3, 4, 5]; delete(a, 1, 4); len(a)", tInt(2))
	checkPrints(t, "a = [1, 2, 3, 4, 5]; delete(a, 1, 4); println(a)", "[1, 5]\n")
	check(t, "a = [1, 2, 3]; delete(a, 0, 3); len(a)", tInt(0)) // delete the whole array
	check(t, "a = [1, 2, 3]; delete(a, 0, 0); len(a)", tInt(3)) // empty range -- nothing removed
	checkPrints(t, "a = [1, 2, 3]; delete(a, 1, 1); println(a)", "[1, 2, 3]\n")

	// mutates in place and returns the same array (identity, not a copy)
	check(t, "a = [1, 2, 3]; b = delete(a, 0); a == b", tBool(true))
	check(t, "a = [1, 2, 3]; b = delete(a, 0); len(b)", tInt(2))
	check(t, "a = [1, 2, 3]; delete(a, 0); len(a)", tInt(2)) // a itself reflects the mutation

	// composes with other array builtins
	checkPrints(t, "println(delete(reverse([1, 2, 3]), 0))", "[2, 1]\n")
}

func TestDeleteOnMaps(t *testing.T) {
	// delete(m, key) removes a key -- the replacement for map_del.
	check(t, "m = {'a': 1, 'b': 2}; delete(m, 'a'); len(m)", tInt(1))
	check(t, "m = {'a': 1, 'b': 2}; delete(m, 'a'); contains(m, 'a')", tBool(false))
	check(t, "m = {'a': 1, 'b': 2}; delete(m, 'a'); m['b']", tInt(2))
	check(t, "m = {7: 'seven'}; delete(m, 7); len(m)", tInt(0)) // int keys too
	check(t, "m = {1.5: 'x'}; delete(m, 1.5); len(m)", tInt(0)) // float keys too
	check(t, "m = {1.5: 'x', 2: 'y'}; delete(m, 1.5); m[2]", tString("y"))

	// a key that was never present is not an error
	check(t, "m = {'a': 1}; delete(m, 'missing'); len(m)", tInt(1))
	check(t, "m = {}; delete(m, 'a'); len(m)", tInt(0))
	check(t, "m = {1.5: 'x'}; delete(m, 2.5); len(m)", tInt(1)) // absent float key

	// mutates in place and returns the same map (identity, not a copy)
	check(t, "m = {'a': 1}; n = delete(m, 'a'); m == n", tBool(true))

	// the key must be an int, float or a string, and the range form is map-invalid
	expectRuntimeError(t, "delete({'a': 1}, true)")
	expectRuntimeError(t, "delete({'a': 1}, [1])")
	expectRuntimeError(t, "delete({'a': 1}, 'a', 2)")
}

func TestDeleteOutOfRange(t *testing.T) {
	expectRuntimeError(t, "delete([1, 2, 3], -1)")    // negative start
	expectRuntimeError(t, "delete([1, 2, 3], 3)")     // start == length
	expectRuntimeError(t, "delete([1, 2, 3], 0, 4)")  // end > length
	expectRuntimeError(t, "delete([1, 2, 3], 0, -1)") // negative end
	expectRuntimeError(t, "delete([1, 2, 3], 2, 1)")  // start > end
	expectRuntimeError(t, "delete([], 0)")            // empty array
}

func TestDeleteWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "delete(123, 0)") // first arg must be an array or a map
	expectRuntimeError(t, "delete('not an array', 0)")
	expectRuntimeError(t, "delete([1, 2, 3], 'a')")    // start must be an int
	expectRuntimeError(t, "delete([1, 2, 3], 0, 'a')") // end must be an int
	expectRuntimeError(t, "delete([1, 2, 3], true)")
}

func TestDeleteArity(t *testing.T) {
	expectRuntimeError(t, "delete([1, 2, 3])")
	expectRuntimeError(t, "delete([1, 2, 3], 0, 1, 2)")
}

func TestReverse(t *testing.T) {
	check(t, "a = reverse([1, 2, 3]); len(a)", tInt(3))
	check(t, "a = reverse([1, 2, 3]); a[0]", tInt(3))
	check(t, "a = reverse([1, 2, 3]); a[1]", tInt(2))
	check(t, "a = reverse([1, 2, 3]); a[2]", tInt(1))

	check(t, "len(reverse([]))", tInt(0))         // empty array
	check(t, "a = reverse([1]); len(a)", tInt(1)) // single element -- unchanged
	check(t, "a = reverse([1]); a[0]", tInt(1))
	check(t, "a = reverse(['a', 'b', 'c']); a[0]", tString("c")) // non-numeric elements
	check(t, "a = reverse([1, 'two', true]); a[0]", tBool(true)) // mixed-type elements preserved

	check(t, "a = reverse(reverse([1, 2, 3])); a[0]", tInt(1)) // reversing twice round-trips

	// the source array is untouched, and the result is a fresh array
	check(t, "a = [1, 2, 3]; b = reverse(a); a[0]", tInt(1))
	check(t, "a = [1, 2, 3]; b = reverse(a); append(b, 4); len(a)", tInt(3))
}

func TestReverseWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "reverse(123)")
	expectRuntimeError(t, "reverse('a string')")
	expectRuntimeError(t, "reverse(true)")
	expectRuntimeError(t, "reverse({})")
}

func TestReverseArity(t *testing.T) {
	expectRuntimeError(t, "reverse()")
	expectRuntimeError(t, "reverse([1, 2, 3], 'extra')")
}

func TestConcat(t *testing.T) {
	check(t, "len(concat())", tInt(0)) // no args -> empty array

	check(t, "a = concat([1, 2, 3]); len(a)", tInt(3)) // single array -> same contents
	check(t, "a = concat([1, 2, 3]); a[0]", tInt(1))
	check(t, "a = concat([1, 2, 3]); a[2]", tInt(3))

	check(t, "a = concat([1, 2], [3, 4]); len(a)", tInt(4))
	check(t, "a = concat([1, 2], [3, 4]); a[0]", tInt(1))
	check(t, "a = concat([1, 2], [3, 4]); a[1]", tInt(2))
	check(t, "a = concat([1, 2], [3, 4]); a[2]", tInt(3))
	check(t, "a = concat([1, 2], [3, 4]); a[3]", tInt(4))

	check(t, "a = concat([1], [2], [3]); len(a)", tInt(3)) // more than two arrays
	check(t, "a = concat([1], [2], [3]); a[1]", tInt(2))

	check(t, "a = concat([], [1, 2]); len(a)", tInt(2)) // leading empty array
	check(t, "a = concat([1, 2], []); len(a)", tInt(2)) // trailing empty array
	check(t, "a = concat([], []); len(a)", tInt(0))     // all empty

	check(t, "a = concat(['a', 1, true]); a[2]", tBool(true)) // mixed-type elements preserved

	// the returned array is a fresh copy -- mutating it doesn't touch the sources
	check(t, "a = [1, 2]; b = concat(a, [3]); append(b, 4); len(a)", tInt(2))
}

func TestConcatWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "concat(123)")
	expectRuntimeError(t, "concat('not an array')")
	expectRuntimeError(t, "concat([1, 2], 123)") // fails on the first non-array
	expectRuntimeError(t, "concat([1, 2], true)")
	expectRuntimeError(t, "concat({}, [1, 2])") // map, not array
}

func TestContainsString(t *testing.T) {
	check(t, "contains('Hello World', 'World')", tBool(true))
	check(t, "contains('Hello World', 'world')", tBool(false)) // case-sensitive
	check(t, "contains('Hello World', '')", tBool(true))       // empty needle always matches
	check(t, "contains('', '')", tBool(true))
	check(t, "contains('', 'a')", tBool(false))
	check(t, "contains('Hello World', 'Hello World')", tBool(true)) // exact match
	check(t, "contains('Hello World', 'xyz')", tBool(false))
}

func TestContainsArray(t *testing.T) {
	check(t, "contains([1, 2, 3], 2)", tBool(true))
	check(t, "contains([1, 2, 3], 5)", tBool(false))
	check(t, "contains([], 1)", tBool(false)) // empty array
	check(t, "contains(['a', 'b', 'c'], 'b')", tBool(true))
	check(t, "contains(['a', 'b', 'c'], 'z')", tBool(false))
	check(t, "contains([1, 2, 3], 2.0)", tBool(true))           // cross-kind numeric comparison
	check(t, "contains([1, 'a', true], 'a')", tBool(true))      // mixed-type array
	check(t, "contains([1, 2, 3], '2')", tBool(false))          // string '2' != int 2, no coercion
	check(t, "contains([[1, 2], [3, 4]], [1, 2])", tBool(true)) // array equality by value
}

// `{a, b}` -- a key with no ': value' -- initializes that key to unit, so a
// map can be pre-shaped with the keys it will hold before it holds them.
func TestMapShorthandKeysInitializeToUnit(t *testing.T) {
	check(t, "{a}['a']", tUnit())
	check(t, "is_typeof({a}['a'], 'unit')", tBool(true))
	check(t, "len({a, b, c})", tInt(3))

	// The key is the identifier's *name*, exactly as in `{'a': ...}` -- it is
	// not evaluated as a variable, so an `a` in scope is irrelevant.
	check(t, "a = 'zzz'; keys({a})[0]", tString("a"))
	check(t, "a = 'zzz'; is_typeof({a}['a'], 'unit')", tBool(true))

	// Non-identifier keys take the shorthand too, and keep their own kind.
	check(t, "is_typeof({1, 2}[1], 'unit')", tBool(true))
	check(t, "is_typeof({'a'}['a'], 'unit')", tBool(true))
	check(t, "len({a,})", tInt(1)) // trailing comma
}

// The bug this guards: evalMapLit's nil-value check was inverted, so a key
// *with* a value got unit stored instead of the value. Mixing both forms in
// one literal is what catches it -- either branch alone looks fine.
func TestMapShorthandAndExplicitValuesMix(t *testing.T) {
	check(t, "{a: 1, b, 'c': 3}['a']", tInt(1))
	check(t, "is_typeof({a: 1, b, 'c': 3}['b'], 'unit')", tBool(true))
	check(t, "{a: 1, b, 'c': 3}['c']", tInt(3))
	check(t, "len({a: 1, b, 'c': 3})", tInt(3))

	// A plain map literal must still be untouched by any of this.
	check(t, "{'a': 1, 'b': 2}['b']", tInt(2))
}

// A duplicate key in a literal is not an error: entries are written left to
// right, so the last one wins and the map holds a single entry for that key.
// This is the same rule as `m[k] = v` twice, which is what evalMapLit does.
func TestMapLiteralDuplicateKeysLastOneWins(t *testing.T) {
	check(t, "{'a': 1, 'a': 2}['a']", tInt(2))
	check(t, "len({'a': 1, 'a': 2})", tInt(1))
	check(t, "{'a': 1, 'a': 2, 'a': 3}['a']", tInt(3))
	check(t, "{1: 'x', 1: 'y'}[1]", tString("y"))

	// An identifier key and the string key of the same name are the same key,
	// so the two forms collide with each other.
	check(t, "{a: 1, 'a': 2}['a']", tInt(2))
	check(t, "{'a': 1, a: 2}['a']", tInt(2))
	check(t, "len({a: 1, 'a': 2})", tInt(1))

	// The shorthand is an entry like any other, so it both overwrites and is
	// overwritten -- a trailing `a` resets an earlier `a: 1` back to unit.
	check(t, "{a, a: 1}['a']", tInt(1))
	check(t, "is_typeof({a: 1, a}['a'], 'unit')", tBool(true))
	check(t, "len({a, a})", tInt(1))

	// Later entries overwrite, but they do not disturb the other keys.
	check(t, "m = {'a': 1, 'b': 2, 'a': 3}; m['b']", tInt(2))
	check(t, "len({'a': 1, 'b': 2, 'a': 3})", tInt(2))
}

// A shorthand key is an ordinary entry once the map exists: assigning to it
// overwrites the unit it was initialized with, rather than adding a key.
func TestMapShorthandKeysAreWritable(t *testing.T) {
	check(t, "m = {name, age}; m['name'] = 'Marcos'; m['name']", tString("Marcos"))
	check(t, "m = {name, age}; m['name'] = 'Marcos'; len(m)", tInt(2))
	check(t, "m = {name}; m.name = 'Marcos'; m.name", tString("Marcos"))
}

func TestContainsMap(t *testing.T) {
	check(t, "m = {'a': 1, 'b': 2}; contains(m, 'a')", tBool(true))
	check(t, "m = {'a': 1, 'b': 2}; contains(m, 'z')", tBool(false))
	check(t, "m = {}; contains(m, 'a')", tBool(false)) // empty map
	check(t, "m = {1: 'x', 2: 'y'}; contains(m, 1)", tBool(true))
	check(t, "m = {1: 'x', 2: 'y'}; contains(m, 3)", tBool(false))
	check(t, "m = {'a': 1}; contains(m, 'A')", tBool(false)) // case-sensitive key
}

func TestContainsWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "contains(123, 1)") // first arg must be string/array/map
	expectRuntimeError(t, "contains(true, 1)")
	expectRuntimeError(t, "contains('abc', 1)") // needle must be a string when haystack is a string
	expectRuntimeError(t, "contains({}, true)") // map key must be string/int
	expectRuntimeError(t, "contains({}, [1])")
}

func TestContainsArity(t *testing.T) {
	expectRuntimeError(t, "contains([1, 2, 3])")
	expectRuntimeError(t, "contains([1, 2, 3], 1, 'extra')")
}

func TestFilter(t *testing.T) {
	check(t, "a = filter([1, 2, 3, 4, 5], \\(x) -> x > 2); len(a)", tInt(3))
	check(t, "a = filter([1, 2, 3, 4, 5], \\(x) -> x > 2); a[0]", tInt(3))
	check(t, "a = filter([1, 2, 3, 4, 5], \\(x) -> x > 2); a[1]", tInt(4))
	check(t, "a = filter([1, 2, 3, 4, 5], \\(x) -> x > 2); a[2]", tInt(5))

	check(t, "a = filter([1, 2, 3], \\(x) -> x > 100); len(a)", tInt(0)) // nothing passes
	check(t, "a = filter([1, 2, 3], \\(x) -> true); len(a)", tInt(3))    // everything passes
	check(t, "a = filter([], \\(x) -> true); len(a)", tInt(0))           // empty input

	check(t, "a = [1, 2, 3]; b = filter(a, \\(x) -> x > 1); len(a)", tInt(3)) // source array untouched

	check(t, "a = filter(['a', 'bb', 'ccc'], \\(s) -> len(s) > 1); len(a)", tInt(2))
	check(t, "a = filter(['a', 'bb', 'ccc'], \\(s) -> len(s) > 1); a[0]", tString("bb"))

	// the closure is a real closure -- it can reference variables captured
	// from its defining scope, not just its own parameter.
	check(t, "threshold = 3; a = filter([1, 2, 3, 4, 5], \\(x) -> x > threshold); len(a)", tInt(2))
}

func TestFilterWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "filter(123, \\(x) -> true)")      // first arg must be an array
	expectRuntimeError(t, "filter([1, 2, 3], 123)")          // second arg must be a function
	expectRuntimeError(t, "filter([1, 2, 3], \\() -> true)") // closure must take exactly one argument
	expectRuntimeError(t, "filter([1, 2, 3], \\(x, y) -> true)")
}

func TestFilterArity(t *testing.T) {
	expectRuntimeError(t, "filter([1, 2, 3])")
	expectRuntimeError(t, "filter([1, 2, 3], \\(x) -> true, 'extra')")
}

func TestMap(t *testing.T) {
	check(t, "a = map([1, 2, 3], \\(x) -> x * 2); len(a)", tInt(3))
	check(t, "a = map([1, 2, 3], \\(x) -> x * 2); a[0]", tInt(2))
	check(t, "a = map([1, 2, 3], \\(x) -> x * 2); a[1]", tInt(4))
	check(t, "a = map([1, 2, 3], \\(x) -> x * 2); a[2]", tInt(6))

	check(t, "a = map([], \\(x) -> x * 2); len(a)", tInt(0)) // empty input

	check(t, "a = [1, 2, 3]; b = map(a, \\(x) -> x * 2); a[0]", tInt(1)) // source array untouched

	// the closure's return type doesn't need to match the input element's
	// type -- map() doesn't constrain it either way.
	check(t, "a = map([1, 2, 3], \\(x) -> x > 1); a[0]", tBool(false))
	check(t, "a = map([1, 2, 3], \\(x) -> x > 1); a[1]", tBool(true))

	// the closure is a real closure -- it can reference variables captured
	// from its defining scope, not just its own parameter.
	check(t, "factor = 10; a = map([1, 2, 3], \\(x) -> x * factor); a[2]", tInt(30))

	// composes with filter()
	check(t, "a = filter(map([1, 2, 3, 4], \\(x) -> x * 2), \\(x) -> x > 4); len(a)", tInt(2))
	check(t, "a = filter(map([1, 2, 3, 4], \\(x) -> x * 2), \\(x) -> x > 4); a[0]", tInt(6))
	check(t, "a = filter(map([1, 2, 3, 4], \\(x) -> x * 2), \\(x) -> x > 4); a[1]", tInt(8))
}

func TestMapWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "map(123, \\(x) -> x)")      // first arg must be an array
	expectRuntimeError(t, "map([1, 2, 3], 123)")       // second arg must be a function
	expectRuntimeError(t, "map([1, 2, 3], \\() -> 1)") // closure must take exactly one argument
	expectRuntimeError(t, "map([1, 2, 3], \\(x, y) -> x)")
}

func TestMapArity(t *testing.T) {
	expectRuntimeError(t, "map([1, 2, 3])")
	expectRuntimeError(t, "map([1, 2, 3], \\(x) -> x, 'extra')")
}

func TestStrings(t *testing.T) {
	check(t, "a = 'Hello World'; a[0]", tString("H"))
	check(t, "'Hello World'[6]", tString("W"))
}

// The string-manipulation builtins (repeat, replace, upper, join, split,
// ord, chr, format, ...) moved to the 'string' native package; their
// tests live in internal/packages/string. (The old string.select builtin was
// replaced by the range operator s[from:to]; see TestRangeExpression.)

func TestPostfixChainsAndMethodCalls(t *testing.T) {
	check(t, `m = {'fn': \(x) -> x * 2}; m.fn(10)`, tInt(20))
	check(t, "m = {}; m.cursor = 0; m.cursor += 1; m.cursor", tInt(1))
	check(t, "m = {}; m.cursor = 0; m.cursor -= 1; m.cursor", tInt(-1))
	check(t, `f = \() -> [10, 20, 30]; f()[1]`, tInt(20))
}

func TestImport(t *testing.T) {
	// TODO: as long as I remember, this should work on windows too
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/utils.mca", []byte("{'double': \\(x) -> x * 2}"), 0o644); err != nil {
		t.Fatal(err)
	}

	mainSrc := "u = import('./utils.mca'); u.double(21)"
	if err := os.WriteFile(dir+"/main.mca", []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(dir + "/main.mca")
	if err != nil {
		t.Fatal(err)
	}

	l := lexer.New(dir+"/main.mca", string(content))
	toks := l.Tokenize()
	if len(l.Errors) > 0 {
		t.Fatalf("lex errors: %v", l.Errors)
	}

	prog := parser.Parse(dir+"/main.mca", toks)
	if len(prog.Errors) > 0 {
		t.Fatalf("parse errors: %v", prog.Errors)
	}

	in := newTestInterp()
	got, err := in.Run(prog.Stmts)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}

	if got.Kind() != KInt || intOf(got) != 42 {
		t.Fatalf("import result = %+v, want int(42)", got)
	}
}

// runHelpCapturingOutput runs src (expected to be one or more help() calls)
// and returns whatever it wrote to stdout, for asserting on the printed
// documentation itself rather than just the (always-unit) return value.
func runHelpCapturingOutput(t *testing.T, src string) string {
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

	var buf bytes.Buffer
	in := New()
	in.Out = &buf
	in.Err = io.Discard

	if _, err := in.Run(prog.Stmts); err != nil {
		t.Fatalf("%q: unexpected runtime error: %v", src, err)
	}

	return buf.String()
}

func TestHelp(t *testing.T) {
	check(t, "help(map)", tUnit())
	check(t, "help()", tUnit())

	out := runHelpCapturingOutput(t, "help(map)")
	if !strings.Contains(out, "map(arr: array, fn: fn) -> array") {
		t.Errorf("help(map) missing its signature line, got:\n%s", out)
	}
	if !strings.Contains(out, "Examples:") {
		t.Errorf("help(map) missing an examples section, got:\n%s", out)
	}
	if !strings.Contains(out, "map([1, 2, 3]") {
		t.Errorf("help(map) missing its example call, got:\n%s", out)
	}

	overview := runHelpCapturingOutput(t, "help()")
	if !strings.Contains(overview, "Math:") || !strings.Contains(overview, "Arrays:") {
		t.Errorf("help() overview missing expected category headers, got:\n%s", overview)
	}
	if !strings.Contains(overview, "map") || !strings.Contains(overview, "concat") {
		t.Errorf("help() overview missing expected builtin names, got:\n%s", overview)
	}
	if !strings.Contains(overview, "run help('name')") {
		t.Errorf("help() overview missing the pointer to help('name'), got:\n%s", overview)
	}
}

func TestHelpUnknownFunctionIsRuntimeError(t *testing.T) {
	expectRuntimeError(t, "help(this_builtin_does_not_exist)")
}

func TestHelpWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "help(123)")
	expectRuntimeError(t, "help(true)")
	expectRuntimeError(t, "help(['map'])")
}

func TestHelpArity(t *testing.T) {
	expectRuntimeError(t, "help('map', 'filter')")
}

// TestHelpDocsCoverAllBuiltins keeps helpDocs honest as builtins get added
// or removed: every registered builtin must have a help entry, and every
// help entry must correspond to a real, still-registered builtin.
func TestHelpDocsCoverAllBuiltins(t *testing.T) {
	for name := range builtins {
		if _, ok := helpDocs[name]; !ok {
			t.Errorf("builtin %q is registered but has no help() entry", name)
		}
	}

	for name := range helpDocs {
		if _, ok := builtins[name]; !ok {
			t.Errorf("help() entry %q does not correspond to any registered builtin", name)
		}
	}
}

// TestHelpCategoriesCoverAllDocs keeps helpCategories (used by help()'s
// no-argument overview) in sync with helpDocs: every doc entry should
// appear in exactly one category.
func TestHelpCategoriesCoverAllDocs(t *testing.T) {
	seen := map[string]bool{}

	for _, cat := range helpCategories {
		for _, name := range cat.Funcs {
			if seen[name] {
				t.Errorf("%q is listed in more than one help category", name)
			}
			seen[name] = true

			if _, ok := helpDocs[name]; !ok {
				t.Errorf("category %q lists %q but there's no help doc for it", cat.Name, name)
			}
		}
	}

	for name := range helpDocs {
		if !seen[name] {
			t.Errorf("help doc %q is not listed in any help category", name)
		}
	}
}

// These lean on the resolver's slot assignments -- the trickier corners where a
// wrong depth or a reused slot would give the wrong answer.
func TestResolvedSlots(t *testing.T) {
	// self-recursion: the body's own name isn't bound until the assignment
	// finishes, so it rides the by-name fallback.
	check(t, "fact = \\(n) -> if n <= 1 { 1 } else { n * fact(n - 1) }; fact(6)", tInt(720))

	// two closures built from the same literal keep independent captured state.
	check(t, "mk = \\() -> { c = 0; \\() -> (c += 1) }; a = mk(); b = mk(); a(); a(); b(); a() * 10 + b()", tInt(32))

	// a variable read three scopes up still resolves to the same slot.
	check(t, "x = 10; f = \\() -> \\() -> \\() -> x; f()()()", tInt(10))

	// assigning inside a block reuses the outer binding, it doesn't shadow it.
	check(t, "x = 1; if true { x = 2 }; x", tInt(2))

	// assigning a builtin's name binds a fresh local that shadows it.
	check(t, "n = len([1, 2, 3]); len = 100; n + len", tInt(103))

	// mutual recursion: is_odd is a forward reference when is_even is resolved.
	check(t, "is_even = \\(x) -> if x == 0 { true } else { is_odd(x - 1) }; is_odd = \\(x) -> if x == 0 { false } else { is_even(x - 1) }; is_even(10)", tBool(true))
}

// BenchmarkArithLoop drives a nested arithmetic loop. The point of the tagged
// union is that the int intermediates here don't allocate the way boxing them
// into an interface did -- run with -benchmem to see allocs/op.
func BenchmarkArithLoop(b *testing.B) {
	src := `
		total = 0
		for i : 300 {
			for j : 300 {
				total += i * j - j + i % 7
			}
		}
		total
	`
	l := lexer.New("", src)
	toks := l.Tokenize()
	if len(l.Errors) > 0 {
		b.Fatalf("lex errors: %v", l.Errors)
	}
	prog := parser.Parse("", toks)
	if len(prog.Errors) > 0 {
		b.Fatalf("parse errors: %v", prog.Errors)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		in := newTestInterp()
		if _, err := in.Run(prog.Stmts); err != nil {
			b.Fatal(err)
		}
	}
}

// runFile lexes/parses/runs src as though it lived at filename (so import()'s
// caller-relative resolution has a real directory to work from) and returns the
// program's value. It fails the test on any lex, parse, or runtime error.
func runFile(t *testing.T, filename, src string) Value {
	t.Helper()

	l := lexer.New(filename, src)
	toks := l.Tokenize()
	if len(l.Errors) > 0 {
		t.Fatalf("%q: lex errors: %v", src, l.Errors)
	}

	prog := parser.Parse(filename, toks)
	if len(prog.Errors) > 0 {
		t.Fatalf("%q: parse errors: %v", src, prog.Errors)
	}

	in := newTestInterp()
	got, err := in.Run(prog.Stmts)
	if err != nil {
		t.Fatalf("%q: unexpected runtime error: %v", src, err)
	}
	return got
}

// writeModule writes a one-file MCA module whose imported value is the integer
// n, so a test can tell which file an import() resolved to.
func writeModule(t *testing.T, dir, name string, n int) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(strconv.Itoa(n)), 0o644); err != nil {
		t.Fatalf("write module %s/%s: %v", dir, name, err)
	}
}

// interp's own tests can't blank-import the native packages (import cycle), so
// nativeModules starts empty here. This registers one throwaway package so the
// package/qualified-member help paths have something to document. RegisterModule
// panics on a duplicate name, so the name must stay unique to this file.
func init() {
	RegisterModule(&Module{
		Name: "helptestpkg",
		Fns: map[string]*Native{
			"ping": NewNative("helptestpkg.ping", 0, func(in *Interp, c *Call) Value { return UnitV() }),
		},
		Docs: map[string]Doc{
			"ping": {Returns: "unit", Description: "A test-only function.", Examples: []string{"helptestpkg.ping()"}},
		},
	})
}

func helpOutput(t *testing.T, name string) string {
	t.Helper()

	var buf bytes.Buffer
	in := New()
	in.Out = &buf
	in.Err = io.Discard

	if err := in.Help(name); err != nil {
		t.Fatalf("Help(%q) returned error: %v", name, err)
	}
	return buf.String()
}

func TestHelpOverviewNoName(t *testing.T) {
	out := helpOutput(t, "")
	// the overview lists the builtin categories and the importable packages.
	for _, want := range []string{"MCA builtin functions", "Packages", "helptestpkg"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Help(\"\") output missing %q; got:\n%s", want, out)
		}
	}
}

func TestHelpNameDocumentsBuiltinPackageAndMember(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"sort", "sort(arr: array, cmp: fn)"},         // an always-there builtin
		{"helptestpkg", "package 'helptestpkg'"},      // a whole package
		{"helptestpkg.ping", "A test-only function."}, // a qualified member
	}

	for _, c := range cases {
		if got := helpOutput(t, c.name); !strings.Contains(got, c.want) {
			t.Fatalf("Help(%q): output missing %q; got:\n%s", c.name, c.want, got)
		}
	}
}

func TestHelpUnknownNameErrorsWithoutPositionPrefix(t *testing.T) {
	in := New()
	in.Out = io.Discard
	in.Err = io.Discard

	err := in.Help("definitely_not_a_builtin")
	if err == nil {
		t.Fatalf("Help(unknown): expected an error, got nil")
	}
	// the CLI prints this straight, so it must not carry a "0:0: runtime error:"
	// source-position prefix -- there is no source to blame.
	if msg := err.Error(); strings.Contains(msg, "runtime error") || strings.Contains(msg, "0:0") {
		t.Fatalf("Help(unknown): error should have no position prefix, got %q", msg)
	}
}

func TestModuleSearchPathsParsing(t *testing.T) {
	sep := string(os.PathListSeparator)

	t.Setenv("MCA_SEARCH_PATHS", "")
	if got := moduleSearchPaths(); got != nil {
		t.Fatalf("empty env: want nil, got %v", got)
	}

	t.Setenv("MCA_SEARCH_PATHS", "/a")
	if got := moduleSearchPaths(); len(got) != 1 || got[0] != "/a" {
		t.Fatalf("single: want [/a], got %v", got)
	}

	t.Setenv("MCA_SEARCH_PATHS", "/a"+sep+"/b"+sep+"/c")
	if got := moduleSearchPaths(); len(got) != 3 || got[0] != "/a" || got[1] != "/b" || got[2] != "/c" {
		t.Fatalf("multi: want [/a /b /c] in order, got %v", got)
	}

	// empty segments (leading/trailing/double separators) are dropped.
	t.Setenv("MCA_SEARCH_PATHS", sep+"/a"+sep+sep+"/b"+sep)
	if got := moduleSearchPaths(); len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("empty segments: want [/a /b], got %v", got)
	}
}

func TestImportCompleteNameResolvesOnSearchPath(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "json.mca", 42)
	t.Setenv("MCA_SEARCH_PATHS", dir)

	got := runFile(t, filepath.Join(t.TempDir(), "prog.mca"), "import('json.mca')")
	if intOf(got) != 42 {
		t.Fatalf("import('json.mca') = %v, want 42", got)
	}
}

func TestImportSearchPathHonorsOrder(t *testing.T) {
	sep := string(os.PathListSeparator)
	first, second := t.TempDir(), t.TempDir()
	writeModule(t, first, "lib.mca", 1)
	writeModule(t, second, "lib.mca", 2)

	// first directory listed wins even though both hold the file.
	t.Setenv("MCA_SEARCH_PATHS", first+sep+second)
	if got := runFile(t, "prog.mca", "import('lib.mca')"); intOf(got) != 1 {
		t.Fatalf("first-wins: import('lib.mca') = %v, want 1", got)
	}

	// swapping the order swaps the winner.
	t.Setenv("MCA_SEARCH_PATHS", second+sep+first)
	if got := runFile(t, "prog.mca", "import('lib.mca')"); intOf(got) != 2 {
		t.Fatalf("swapped: import('lib.mca') = %v, want 2", got)
	}
}

func TestImportSearchPathSkipsDirsWithoutTheFile(t *testing.T) {
	sep := string(os.PathListSeparator)
	empty, holding := t.TempDir(), t.TempDir()
	writeModule(t, holding, "lib.mca", 7)

	// the first directory lacks the file, so resolution keeps scanning.
	t.Setenv("MCA_SEARCH_PATHS", empty+sep+holding)
	if got := runFile(t, "prog.mca", "import('lib.mca')"); intOf(got) != 7 {
		t.Fatalf("skip-empty: import('lib.mca') = %v, want 7", got)
	}
}

func TestImportRelativePathIgnoresSearchPath(t *testing.T) {
	callerDir := t.TempDir()
	writeModule(t, callerDir, "dep.mca", 10)

	decoy := t.TempDir()
	writeModule(t, decoy, "dep.mca", 20)
	t.Setenv("MCA_SEARCH_PATHS", decoy)

	// './dep.mca' resolves next to the importing file, never the search path.
	got := runFile(t, filepath.Join(callerDir, "main.mca"), "import('./dep.mca')")
	if intOf(got) != 10 {
		t.Fatalf("import('./dep.mca') = %v, want 10 (caller-relative, not the decoy)", got)
	}
}

func TestImportAbsolutePathIgnoresSearchPath(t *testing.T) {
	realDir := t.TempDir()
	writeModule(t, realDir, "dep.mca", 33)
	abs := filepath.Join(realDir, "dep.mca")

	decoy := t.TempDir()
	writeModule(t, decoy, "dep.mca", 44)
	t.Setenv("MCA_SEARCH_PATHS", decoy)

	got := runFile(t, "prog.mca", "import('"+abs+"')")
	if intOf(got) != 33 {
		t.Fatalf("absolute import = %v, want 33 (used as-is, not the decoy)", got)
	}
}

func TestImportCompleteNameFallsBackToWorkingDir(t *testing.T) {
	// with the file absent from every search path, a complete name keeps its
	// old working-directory-relative behavior.
	work := t.TempDir()
	writeModule(t, work, "mod.mca", 99)
	t.Chdir(work)
	t.Setenv("MCA_SEARCH_PATHS", t.TempDir()) // a path that does not hold mod.mca

	if got := runFile(t, "prog.mca", "import('mod.mca')"); intOf(got) != 99 {
		t.Fatalf("fallback: import('mod.mca') = %v, want 99", got)
	}
}

func TestImportCompleteNameWithoutSearchPathEnv(t *testing.T) {
	// unset variable: a complete name resolves working-directory-relative,
	// exactly as it did before the search path existed.
	work := t.TempDir()
	writeModule(t, work, "mod.mca", 5)
	t.Chdir(work)
	t.Setenv("MCA_SEARCH_PATHS", "")

	if got := runFile(t, "prog.mca", "import('mod.mca')"); intOf(got) != 5 {
		t.Fatalf("no-env: import('mod.mca') = %v, want 5", got)
	}
}
