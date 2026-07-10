package interp

import (
	"io"
	"math"
	"os"
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

func TestCallBuiltinFunctions(t *testing.T) {
	check(t, "abs(abs(-1) - 2)", tInt(1))
	check(t, "(abs(-1) * 2) ^ 2", tInt(4))
	check(t, "abs(min(abs(-1), max(-5, -4)) * 1)", tInt(4))
	check(t, "max(abs(-12), 8) * sin(rad(30)) + (16 / 2)", tFloat(14))
	check(t, "PI()", tFloat(math.Pi))
	check(t, "E()", tFloat(math.E))
	check(t, "abs(-15.5)", tFloat(15.5))
	check(t, "max(10.5, 20.0)", tFloat(20.0))
	check(t, "min(10.5, 20.0)", tFloat(10.5))
	check(t, "sin(0)", tInt(0))
	check(t, "deg(asin(1))", tInt(90))
	check(t, "deg(acos(0))", tInt(90))
	check(t, "cos(0)", tInt(1))
	check(t, "tan(0)", tInt(0))
	check(t, "rad(180)", tFloat(math.Pi))
	check(t, "deg(3.14159265358979323846)", tInt(180))
	check(t, "sqrt(25)", tInt(5))
	check(t, "log(1)", tInt(0))
	check(t, "log10(1000)", tInt(3))
	check(t, "exp(1)", tFloat(math.E))
	check(t, "floor(4.8)", tInt(4))
	check(t, "ceil(4.2)", tInt(5))
	check(t, "round(4.5)", tInt(5))
	check(t, "round(4.4)", tInt(4))
	check(t, "type(4.4)", tString("float"))
	check(t, "type(4)", tString("int"))
	check(t, "srand(4)", tUnit())
	check(t, "rand(1, 10)", tInt(2)) // glibc-compatible RNG, verified against the C binary
	check(t, "len('Hello World')", tInt(11))
	checkArgs(t, "argc()", tInt(0))
	checkArgs(t, "argc()", tInt(1), "fakename.mca")
	checkArgs(t, "argv(0)", tString("fakename.mca"), "fakename.mca")
	checkArgs(t, "argv(1)", tString("fakearg"), "fakename.mca", "fakearg")
	check(t, "is_int(1)", tBool(true))
	check(t, "is_int(1.3)", tBool(false))
	check(t, "is_float(1)", tBool(false))
	check(t, "is_float(1.3)", tBool(true))
	check(t, "is_string(1)", tBool(false))
	check(t, "is_string('1.3')", tBool(true))
	check(t, "is_bool(1)", tBool(false))
	check(t, "is_bool(false)", tBool(true))
	check(t, "is_unit(1)", tBool(false))
	check(t, "is_unit(?)", tBool(true))
	check(t, "'Hello, World'[7]", tString("W"))
	check(t, "select('Hello, World', 7, 12)", tString("World"))
	check(t, "select('heyhey', 0, 6)", tString("heyhey"))
	check(t, "select('heyhey', 2, 3)", tString("y"))
	check(t, "select('heyhey', 3, 6)", tString("hey"))
	check(t, "ord('a')", tInt(int64('a')))
	check(t, "ord('b')", tInt(int64('b')))
	check(t, "ord('z')", tInt(int64('z')))
	check(t, "format('Hello ', 'World!', ' I am ', 5, ' years old. And ', 5.6, ' feet. I am a ', true, ' tall. I am not ', false)",
		tString("Hello World! I am 5 years old. And 5.6 feet. I am a true tall. I am not false"))
}

func TestReadEntireFile(t *testing.T) {
	// TODO: fix this path to use a local one
	check(t, "read_entire_file('../../../test/file.txt')", tString("Hello World\n"))
}

func TestPrintingReturnsLastArgument(t *testing.T) {
	check(t, "print()", tUnit())
	check(t, "print(PI())", tFloat(math.Pi))
	check(t, "print(PI(), E(), 10)", tInt(10))
	check(t, "println()", tUnit())
	check(t, "println(PI())", tFloat(math.Pi))
	check(t, "println(PI(), E(), 10)", tInt(10))
}

func TestGlobalVariables(t *testing.T) {
	check(t, "x = 10", tInt(10))
	check(t, "y = x = 10", tInt(10))
	check(t, "y = x = 10;y", tInt(10))
	check(t, "x = 10; y = -5.5; z = abs(x * y); println(x, y, z, x + y + z)", tFloat(59.5))
}

func TestAssignment(t *testing.T) {
	check(t, "y = x = 10; x + y", tInt(20))
	check(t, "i = 0; while i < 10 { i += 1 }", tInt(10))
	check(t, "i = 10; while i > 10 { i -= 1 }", tUnit())
	check(t, "i = 10; i += 2", tInt(12))
	check(t, "i = 10; i += 2; i", tInt(12))
	check(t, "i = 10; i -= 2", tInt(8))
	check(t, "i = 10; i -= 2; i", tInt(8))
	check(t, "m = {}; m['name'] = 'Fred'; m['age'] = 32; format(m['name'], ' is ', m['age'], ' years old')", tString("Fred is 32 years old"))
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
	check(t, "r = while 1 { n = 10; break floor(10 * 10 - cos(45)); println(10); }; r", tInt(99))
}

func TestForLoopBreakIsFixed(t *testing.T) {
	check(t, "r = 0; for i : 10 { if i == 3 { break; }; r = i }; r", tInt(2))
	check(t, "r = for i : 10 { if i == 3 { break 99 } }; r", tInt(99))
}

func TestIfs(t *testing.T) {
	check(t, "x = 10; if x == 10 { x = 11.3 }", tFloat(11.3))
	check(t, "if 0 == 0;", tUnit())
	check(t, "x = if 10 != 10.1 { 1337 }", tInt(1337))
	check(t, "x = if 10 != 10.1 {}", tUnit())
	check(t, "if false 0 elif false 1 else true 2", tInt(2))
	check(t, "if false { 0 } elif false { 1 } else true 2", tInt(2))
	check(t, "if false { 0 } elif false { 1 } else { 2; 3; 4; }", tInt(4))
	check(t, "srand(4); a = if rand(0, 10) % 2 == 0 'Ok' else 'Fail'; println(a)", tString("Ok"))
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
	check(t, "as_string(120.234)", tString("120.234000"))
	check(t, "as_string(-120.234)", tString("-120.234000"))
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

func TestUserVariableShadowsBuiltinAsCallee(t *testing.T) {
	// A bare-identifier callee only falls back to the builtins table when
	// no user variable of that name is in scope -- a same-named variable
	// always wins.
	check(t, `len = \(x) -> 999; len('hi')`, tInt(999))
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
	check(t,
		"m = {'name': 'John Doe','age': 32,'weight': 67.56,'is_dead': false,10: 'test'};"+
			"format(len(m), ';', m['name'], ';', m['age'], ';', m['weight'], ';', m['is_dead'], ';', m[10])",
		tString("5;John Doe;32;67.56;false;test"))
	check(t, "m = {}; len(m);", tInt(0))
	check(t, "m = {}; m['width'] = '3rem'; m['height'] = '3rem'; m['z-index'] = 999; map_del(m, 'height')", tBool(true))
	check(t, "m = {}; m['width'] = '3rem'; m['height'] = '3rem'; m['z-index'] = 999; map_del(m, 'Height')", tBool(false))
	check(t, "m = {}; m['width'] = '3rem'; m['height'] = '3rem'; m['z-index'] = 999; map_clear(m); len(m)", tInt(0))
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

func TestJoin(t *testing.T) {
	check(t, "join(['a', 'b', 'c'], ',')", tString("a,b,c"))
	check(t, "join(['a', 'b', 'c'], '')", tString("abc"))
	check(t, "join(['a', 'b', 'c'], ' -- ')", tString("a -- b -- c"))
	check(t, "join(['solo'], ',')", tString("solo"))
	check(t, "join([], ',')", tString(""))
}

func TestJoinRejectsNonStringItemsAndWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "join([1, 2, 3], ',')")      // ints, not strings
	expectRuntimeError(t, "join(['a', 2, 'c'], ',')")  // mixed -- fails on the first non-string
	expectRuntimeError(t, "join('not an array', ',')") // first arg must be an array
	expectRuntimeError(t, "join(['a', 'b'], 1)")       // separator must be a string
}

func TestJoinArity(t *testing.T) {
	expectRuntimeError(t, "join(['a', 'b'])")
	expectRuntimeError(t, "join(['a', 'b'], ',', 'extra')")
}

func TestSplit(t *testing.T) {
	check(t, "s = split('a,b,c', ','); len(s)", tInt(3))
	check(t, "s = split('a,b,c', ','); s[0]", tString("a"))
	check(t, "s = split('a,b,c', ','); s[1]", tString("b"))
	check(t, "s = split('a,b,c', ','); s[2]", tString("c"))
	check(t, "join(split('a,b,c', ','), ',')", tString("a,b,c")) // round-trips through join

	check(t, "s = split('a::b::c', '::'); len(s)", tInt(3)) // multi-char separator
	check(t, "s = split('a::b::c', '::'); s[1]", tString("b"))

	check(t, "s = split('hello', ','); len(s)", tInt(1)) // separator not present -> whole string, unsplit
	check(t, "s = split('hello', ','); s[0]", tString("hello"))

	check(t, "s = split('', ','); len(s)", tInt(1)) // empty input -> single empty-string element
	check(t, "s = split('', ','); s[0]", tString(""))

	check(t, "s = split('a,,b', ','); len(s)", tInt(3)) // consecutive separators -> empty element between them
	check(t, "s = split('a,,b', ','); s[1]", tString(""))

	check(t, "s = split(',a,', ','); len(s)", tInt(3)) // leading/trailing separators -> empty elements at the ends
	check(t, "s = split(',a,', ','); s[0]", tString(""))
	check(t, "s = split(',a,', ','); s[2]", tString(""))

	check(t, "s = split('abc', ''); len(s)", tInt(3)) // empty separator -> split between every rune
	check(t, "s = split('abc', ''); s[0]", tString("a"))
	check(t, "s = split('abc', ''); s[2]", tString("c"))
}

func TestSplitWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "split(123, ',')")   // first arg must be a string
	expectRuntimeError(t, "split('a,b', 123)") // separator must be a string
	expectRuntimeError(t, "split(['a'], ',')")
}

func TestSplitArity(t *testing.T) {
	expectRuntimeError(t, "split('a,b')")
	expectRuntimeError(t, "split('a,b', ',', 'extra')")
}

func TestChr(t *testing.T) {
	check(t, "chr(65)", tString("A"))
	check(t, "chr(97)", tString("a"))
	check(t, "chr(48)", tString("0"))
	check(t, "chr(32)", tString(" "))

	check(t, "chr(128512)", tString("\U0001F600")) // multi-byte code point (an emoji)
	check(t, "len(chr(128512))", tInt(4))          // ... encoded as 4 UTF-8 bytes

	// round-trips through ord() for single-byte code points (ord() indexes
	// by byte, not rune, so this doesn't hold for multi-byte code points).
	check(t, "ord(chr(65))", tInt(65))
	check(t, "chr(ord('Q'))", tString("Q"))
}

func TestChrOnInvalidCodepoints(t *testing.T) {
	// Out-of-range code points (negative, or beyond the max valid rune
	// 0x10FFFF) fall back to the Unicode replacement character (U+FFFD)
	// rather than erroring, matching Go's string(rune(...)) conversion.
	check(t, "chr(-1)", tString("�"))
	check(t, "chr(2000000)", tString("�"))
	check(t, "len(chr(-1))", tInt(3))
	check(t, "len(chr(2000000))", tInt(3))
}

func TestChrWrongArgTypes(t *testing.T) {
	expectRuntimeError(t, "chr('a')")
	expectRuntimeError(t, "chr(1.5)")
	expectRuntimeError(t, "chr(true)")
	expectRuntimeError(t, "chr([65])")
}

func TestChrArity(t *testing.T) {
	expectRuntimeError(t, "chr()")
	expectRuntimeError(t, "chr(65, 66)")
}

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
