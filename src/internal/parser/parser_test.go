package parser

import (
	"os"
	"path/filepath"
	"testing"

	"mca/internal/ast"
	"mca/internal/lexer"
)

func parseSrc(t *testing.T, src string) *Program {
	t.Helper()

	l := lexer.New("test.mca", src)
	toks := l.Tokenize()

	if len(l.Errors) > 0 {
		t.Fatalf("lex errors for %q: %v", src, l.Errors)
	}

	return Parse("test.mca", toks)
}

func mustParseOK(t *testing.T, src string) *Program {
	t.Helper()

	prog := parseSrc(t, src)
	if len(prog.Errors) > 0 {
		t.Fatalf("unexpected parse errors for %q: %v", src, prog.Errors)
	}

	return prog
}

func TestExamplesAllParseCleanly(t *testing.T) {
	root := "../../../examples" // TODO: later, we're going to make the go version the primary one

	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".mca" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking examples dir: %v", err)
	}

	if len(files) == 0 {
		t.Fatalf("no example files found under %s", root)
	}

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		l := lexer.New(path, string(content))
		toks := l.Tokenize()
		if len(l.Errors) > 0 {
			t.Fatalf("%s: lex errors: %v", path, l.Errors)
		}

		prog := Parse(path, toks)
		if len(prog.Errors) > 0 {
			t.Fatalf("%s: parse errors: %v", path, prog.Errors)
		}
	}
}

func TestRightAssociativePower(t *testing.T) {
	// 2^3^2 should parse as 2^(3^2), TODO: but I think it should be ((2^3)^2) which gives a completely different value
	prog := mustParseOK(t, "2^3^2")

	bin, ok := prog.Stmts[0].(*ast.BinaryExpr)
	if !ok || bin.Op != ast.PowOp {
		t.Fatalf("expected top-level PowOp, got %#v", prog.Stmts[0])
	}

	rightBin, ok := bin.Right.(*ast.BinaryExpr)
	if !ok || rightBin.Op != ast.PowOp {
		t.Fatalf("expected right-associative nesting, got right=%#v", bin.Right)
	}

	if _, ok := bin.Left.(*ast.IntLit); !ok {
		t.Fatalf("expected left operand to be the outermost IntLit(2), got %#v", bin.Left)
	}
}

func TestShiftPrecedence(t *testing.T) {
	// C-style precedence: shifts bind looser than '+' and tighter than '<',
	// so 1 + 2 << 3 < 4 is ((1 + 2) << 3) < 4.
	prog := mustParseOK(t, "1 + 2 << 3 < 4")

	cmp, ok := prog.Stmts[0].(*ast.BinaryExpr)
	if !ok || cmp.Op != ast.LtOp {
		t.Fatalf("expected top-level LtOp, got %#v", prog.Stmts[0])
	}

	shift, ok := cmp.Left.(*ast.BinaryExpr)
	if !ok || shift.Op != ast.ShlOp {
		t.Fatalf("expected ShlOp under '<', got %#v", cmp.Left)
	}

	if add, ok := shift.Left.(*ast.BinaryExpr); !ok || add.Op != ast.PlusOp {
		t.Fatalf("expected PlusOp under '<<', got %#v", shift.Left)
	}

	// Shifts chain left-associatively: a << b >> c is (a << b) >> c.
	prog = mustParseOK(t, "a << b >> c")

	outer, ok := prog.Stmts[0].(*ast.BinaryExpr)
	if !ok || outer.Op != ast.ShrOp {
		t.Fatalf("expected top-level ShrOp, got %#v", prog.Stmts[0])
	}

	if inner, ok := outer.Left.(*ast.BinaryExpr); !ok || inner.Op != ast.ShlOp {
		t.Fatalf("expected left-associative ShlOp on the left, got %#v", outer.Left)
	}
}

func TestBitwisePrecedence(t *testing.T) {
	// Lua-style ladder: relational < | < ~ < & < shift, so
	// a | b ~ c & d << e parses as a | (b ~ (c & (d << e))).
	prog := mustParseOK(t, "a | b ~ c & d << e")

	or, ok := prog.Stmts[0].(*ast.BinaryExpr)
	if !ok || or.Op != ast.BitOrOp {
		t.Fatalf("expected top-level BitOrOp, got %#v", prog.Stmts[0])
	}

	xor, ok := or.Right.(*ast.BinaryExpr)
	if !ok || xor.Op != ast.XorOp {
		t.Fatalf("expected XorOp under '|', got %#v", or.Right)
	}

	band, ok := xor.Right.(*ast.BinaryExpr)
	if !ok || band.Op != ast.BitAndOp {
		t.Fatalf("expected BitAndOp under '~', got %#v", xor.Right)
	}

	if shl, ok := band.Right.(*ast.BinaryExpr); !ok || shl.Op != ast.ShlOp {
		t.Fatalf("expected ShlOp under '&', got %#v", band.Right)
	}

	// ... and all of them bind tighter than a comparison, unlike C:
	// a ~ b == c is (a ~ b) == c.
	prog = mustParseOK(t, "a ~ b == c")

	eq, ok := prog.Stmts[0].(*ast.BinaryExpr)
	if !ok || eq.Op != ast.EqualOp {
		t.Fatalf("expected top-level EqualOp, got %#v", prog.Stmts[0])
	}

	if left, ok := eq.Left.(*ast.BinaryExpr); !ok || left.Op != ast.XorOp {
		t.Fatalf("expected XorOp on the left of '==', got %#v", eq.Left)
	}
}

func TestTildeIsUnaryOrBinaryByPosition(t *testing.T) {
	// prefix '~' is bitwise not ...
	prog := mustParseOK(t, "~x")
	un, ok := prog.Stmts[0].(*ast.UnaryExpr)
	if !ok || un.Op != ast.BitNotOp {
		t.Fatalf("expected prefix BitNotOp, got %#v", prog.Stmts[0])
	}

	// ... an infix '~' is xor, and both can appear in one expression.
	prog = mustParseOK(t, "a ~ ~b")
	bin, ok := prog.Stmts[0].(*ast.BinaryExpr)
	if !ok || bin.Op != ast.XorOp {
		t.Fatalf("expected binary XorOp, got %#v", prog.Stmts[0])
	}
	if right, ok := bin.Right.(*ast.UnaryExpr); !ok || right.Op != ast.BitNotOp {
		t.Fatalf("expected BitNotOp as the right operand, got %#v", bin.Right)
	}
}

func TestPrefixOperatorsDoNotStack(t *testing.T) {
	// TODO: make it possible later
	prog := parseSrc(t, "!!x")
	if len(prog.Errors) == 0 {
		t.Fatalf("expected a parse error for stacked prefix '!!x'")
	}

	prog = parseSrc(t, "--x")
	if len(prog.Errors) == 0 {
		t.Fatalf("expected a parse error for stacked prefix '--x'")
	}

	prog = parseSrc(t, "~~x")
	if len(prog.Errors) == 0 {
		t.Fatalf("expected a parse error for stacked prefix '~~x'")
	}
}

func TestFactorialVsNotDisambiguation(t *testing.T) {
	prog := mustParseOK(t, "5!")
	un, ok := prog.Stmts[0].(*ast.UnaryExpr)
	if !ok || un.Op != ast.FactorialOp {
		t.Fatalf("expected postfix FactorialOp, got %#v", prog.Stmts[0])
	}

	prog = mustParseOK(t, "!true")
	un, ok = prog.Stmts[0].(*ast.UnaryExpr)
	if !ok || un.Op != ast.NotOp {
		t.Fatalf("expected prefix NotOp, got %#v", prog.Stmts[0])
	}
}

func TestAssignmentIsRightAssociative(t *testing.T) {
	prog := mustParseOK(t, "a = b = 1")

	outer, ok := prog.Stmts[0].(*ast.AssignExpr)
	if !ok {
		t.Fatalf("expected AssignExpr, got %#v", prog.Stmts[0])
	}

	if _, ok := outer.Right.(*ast.AssignExpr); !ok {
		t.Fatalf("expected right-associative nested assignment, got %#v", outer.Right)
	}
}

func TestCompoundAssignOps(t *testing.T) {
	prog := mustParseOK(t, "a += 1")
	assign := prog.Stmts[0].(*ast.AssignExpr)
	if assign.Op != ast.AddAssign {
		t.Fatalf("expected AddAssign, got %v", assign.Op)
	}

	prog = mustParseOK(t, "a -= 1")
	assign = prog.Stmts[0].(*ast.AssignExpr)
	if assign.Op != ast.SubAssign {
		t.Fatalf("expected SubAssign, got %v", assign.Op)
	}
}

func TestCallRecognizedForAnyPostfixExpression(t *testing.T) {
	// '(' is a general postfix operator now: arr[0](5) parses as a single
	// call whose callee is the index expression, not two separate
	// statements.
	prog := mustParseOK(t, "arr[0](5)")

	if len(prog.Stmts) != 1 {
		t.Fatalf("expected 1 top-level statement, got %d: %#v", len(prog.Stmts), prog.Stmts)
	}

	call, ok := prog.Stmts[0].(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected a CallExpr, got %#v", prog.Stmts[0])
	}

	if _, ok := call.Callee.(*ast.SquareExpr); !ok {
		t.Fatalf("expected callee to be a SquareExpr (arr[0]), got %#v", call.Callee)
	}

	if len(call.Args) != 1 {
		t.Fatalf("expected 1 argument, got %d: %#v", len(call.Args), call.Args)
	}

	if lit, ok := call.Args[0].(*ast.IntLit); !ok || lit.Value != 5 {
		t.Fatalf("expected argument to be IntLit(5), got %#v", call.Args[0])
	}
}

func TestCallRecognizedOnParenthesizedFnLiteral(t *testing.T) {
	// The IIFE pattern: (\() -> 1)() must parse as a single call whose
	// callee is the parenthesized fn literal.
	prog := mustParseOK(t, "(\\() -> 1)()")

	call, ok := prog.Stmts[0].(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected a CallExpr, got %#v", prog.Stmts[0])
	}

	if _, ok := call.Callee.(*ast.FnExpr); !ok {
		t.Fatalf("expected callee to be a FnExpr, got %#v", call.Callee)
	}

	if len(call.Args) != 0 {
		t.Fatalf("expected 0 arguments, got %d: %#v", len(call.Args), call.Args)
	}
}

func TestCallRecognizedOnDotField(t *testing.T) {
	// m.f(1, 2) parses as CallExpr{Callee: DotExpr{Left: m, Index: f}}, not
	// a special dot-call node.
	prog := mustParseOK(t, "m.f(1, 2)")

	call, ok := prog.Stmts[0].(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected a CallExpr, got %#v", prog.Stmts[0])
	}

	dot, ok := call.Callee.(*ast.DotExpr)
	if !ok {
		t.Fatalf("expected callee to be a DotExpr, got %#v", call.Callee)
	}

	if ident, ok := dot.Index.(*ast.Ident); !ok || ident.Name != "f" {
		t.Fatalf("expected DotExpr.Index to be Ident(f), got %#v", dot.Index)
	}

	if len(call.Args) != 2 {
		t.Fatalf("expected 2 arguments, got %d: %#v", len(call.Args), call.Args)
	}
}

func TestChainedCalls(t *testing.T) {
	// f()() -- calling the result of a call -- must parse as nested
	// CallExprs, each wrapping the previous one as its callee.
	prog := mustParseOK(t, "f()()")

	outer, ok := prog.Stmts[0].(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected outer stmt to be a CallExpr, got %#v", prog.Stmts[0])
	}

	inner, ok := outer.Callee.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected outer callee to be a CallExpr, got %#v", outer.Callee)
	}

	if ident, ok := inner.Callee.(*ast.Ident); !ok || ident.Name != "f" {
		t.Fatalf("expected innermost callee to be Ident(f), got %#v", inner.Callee)
	}
}

func TestDotPostfixChain(t *testing.T) {
	prog := mustParseOK(t, "a.b.c")

	outer, ok := prog.Stmts[0].(*ast.DotExpr)
	if !ok {
		t.Fatalf("expected IndexExpr, got %#v", prog.Stmts[0])
	}

	inner, ok := outer.Left.(*ast.DotExpr)
	if !ok {
		t.Fatalf("expected nested IndexExpr for a.b, got %#v", outer.Left)
	}

	if id, ok := inner.Left.(*ast.Ident); !ok || id.Name != "a" {
		t.Fatalf("expected innermost left to be Ident(a), got %#v", inner.Left)
	}
}

func TestCurlyIsAlwaysMapOutsideBlockPositions(t *testing.T) {
	prog := mustParseOK(t, "x = {1: 2, 3: 4}")

	assign := prog.Stmts[0].(*ast.AssignExpr)
	m, ok := assign.Right.(*ast.MapExpr)
	if !ok {
		t.Fatalf("expected map literal on assignment RHS, got %#v", assign.Right)
	}

	if len(m.Keys) != 2 || len(m.Values) != 2 {
		t.Fatalf("expected 2 map entries, got %d keys %d values", len(m.Keys), len(m.Values))
	}
}

// A key written without a ': value' parses with a *nil* value expression,
// which is what evalMapLit reads to mean "initialize this one to unit". The
// nil is load-bearing, so it is asserted here rather than just the entry
// count: a shorthand key that came back with some placeholder expression
// instead would evaluate to that placeholder, not to '?'.
func TestMapShorthandKeyParsesWithNilValue(t *testing.T) {
	prog := mustParseOK(t, "x = {a, b: 2, 'c', 3}")

	m, ok := prog.Stmts[0].(*ast.AssignExpr).Right.(*ast.MapExpr)
	if !ok {
		t.Fatalf("expected map literal on assignment RHS, got %#v", prog.Stmts[0])
	}

	if len(m.Keys) != 4 || len(m.Values) != 4 {
		t.Fatalf("expected 4 map entries, got %d keys %d values", len(m.Keys), len(m.Values))
	}

	// Keys and Values stay index-aligned: a shorthand key must not shift the
	// entries after it, or `b` would pick up 2 as *its* key.
	for i, wantNil := range []bool{true, false, true, true} {
		if gotNil := m.Values[i] == nil; gotNil != wantNil {
			t.Errorf("entry %d: value is nil = %v, want %v (value %#v)", i, gotNil, wantNil, m.Values[i])
		}
	}

	if id, ok := m.Keys[1].(*ast.Ident); !ok || id.Name != "b" {
		t.Errorf("expected key 1 to be Ident(b), got %#v", m.Keys[1])
	}
}

func TestMapShorthandKeyAcceptsTrailingComma(t *testing.T) {
	for _, src := range []string{"x = {a}", "x = {a,}", "x = {a, b,}", "x = {a: 1, b,}"} {
		prog := mustParseOK(t, src)

		if _, ok := prog.Stmts[0].(*ast.AssignExpr).Right.(*ast.MapExpr); !ok {
			t.Errorf("%q: expected a map literal, got %#v", src, prog.Stmts[0])
		}
	}
}

// A value is optional now, but a separator is not: a key must be followed by
// ':', ',' or '}'.
func TestMapKeyWithoutSeparatorIsAParseError(t *testing.T) {
	for _, src := range []string{"x = {a 1}", "x = {a: 1 b: 2}", "x = {a b}"} {
		if prog := parseSrc(t, src); len(prog.Errors) == 0 {
			t.Errorf("%q: expected a parse error, got none", src)
		}
	}
}

func TestBlockAcceptsBareInlineOrBraced(t *testing.T) {
	prog := mustParseOK(t, "while true break;")
	w := prog.Stmts[0].(*ast.WhileExpr)
	if len(w.Body) != 1 {
		t.Fatalf("expected inline single-expression body, got %d stmts", len(w.Body))
	}

	prog = mustParseOK(t, "while true { 1; 2 }")
	w = prog.Stmts[0].(*ast.WhileExpr)
	if len(w.Body) != 2 {
		t.Fatalf("expected 2-statement braced body, got %d", len(w.Body))
	}

	prog = mustParseOK(t, "while true;")
	w = prog.Stmts[0].(*ast.WhileExpr)
	if w.Body != nil {
		t.Fatalf("expected empty body for bare ';' block, got %#v", w.Body)
	}
}

func TestForRangeThreeShapes(t *testing.T) {
	prog := mustParseOK(t, "for i : 10 { i }")
	fr := prog.Stmts[0].(*ast.ForRangeExpr)
	if fr.To != nil || fr.By != nil {
		t.Fatalf("expected bare-count form to leave To/By nil, got To=%#v By=%#v", fr.To, fr.By)
	}

	prog = mustParseOK(t, "for i : [1, 10] { i }")
	fr = prog.Stmts[0].(*ast.ForRangeExpr)
	if fr.To == nil || fr.By != nil {
		t.Fatalf("expected [from,to] form to set To and leave By nil, got To=%#v By=%#v", fr.To, fr.By)
	}

	prog = mustParseOK(t, "for i : [1, 10, 2] { i }")
	fr = prog.Stmts[0].(*ast.ForRangeExpr)
	if fr.To == nil || fr.By == nil {
		t.Fatalf("expected [from,to,by] form to set both, got To=%#v By=%#v", fr.To, fr.By)
	}
}

func TestForOfShape(t *testing.T) {
	prog := mustParseOK(t, "for k, v : arr { k }")
	fo := prog.Stmts[0].(*ast.ForOfExpr)
	if fo.Key.Name != "k" || fo.Value.Name != "v" {
		t.Fatalf("unexpected key/value idents: %#v", fo)
	}
}

func TestIfElifElse(t *testing.T) {
	prog := mustParseOK(t, "if a { 1 } elif b { 2 } elif c { 3 } else { 4 }")
	ifExpr := prog.Stmts[0].(*ast.IfExpr)

	if len(ifExpr.Elifs) != 2 {
		t.Fatalf("expected 2 elif blocks, got %d", len(ifExpr.Elifs))
	}
	if ifExpr.Else == nil {
		t.Fatalf("expected an else block")
	}
}

func TestMissingSemiOnBreakIsReportedAsSuch(t *testing.T) {
	// `break` followed by something that can't start a primary expression
	// (here ')') should hint at a missing ';' rather than a generic parse error.
	prog := parseSrc(t, "while true { break) }")
	if len(prog.Errors) == 0 {
		t.Fatalf("expected a parse error")
	}
}

// commaTargets is a small helper: it asserts the statement is a plain '='
// assignment whose left side is an ArrayExpr of the given identifier names,
// which is how comma-syntax 'a, b = ...' is represented.
func commaTargets(t *testing.T, stmt ast.Expr, want ...string) *ast.AssignExpr {
	t.Helper()

	assign, ok := stmt.(*ast.AssignExpr)
	if !ok || assign.Op != ast.Assign {
		t.Fatalf("expected a plain AssignExpr, got %#v", stmt)
	}

	arr, ok := assign.Left.(*ast.ArrayExpr)
	if !ok {
		t.Fatalf("expected comma targets to be an ArrayExpr, got %#v", assign.Left)
	}

	if len(arr.Items) != len(want) {
		t.Fatalf("expected %d targets, got %d: %#v", len(want), len(arr.Items), arr.Items)
	}

	for i, name := range want {
		id, ok := arr.Items[i].(*ast.Ident)
		if !ok || id.Name != name {
			t.Fatalf("target %d: expected Ident(%s), got %#v", i, name, arr.Items[i])
		}
	}

	return assign
}

func TestCommaSyntaxDestructuring(t *testing.T) {
	// The RHS is a single expression (an array literal); the LHS is the
	// comma list of targets.
	prog := mustParseOK(t, "a, b, c = [call(), 3, 4]")

	assign := commaTargets(t, prog.Stmts[0], "a", "b", "c")

	rhs, ok := assign.Right.(*ast.ArrayExpr)
	if !ok {
		t.Fatalf("expected the RHS to be an ArrayExpr, got %#v", assign.Right)
	}
	if len(rhs.Items) != 3 {
		t.Fatalf("expected 3 items on the RHS, got %d: %#v", len(rhs.Items), rhs.Items)
	}
	if _, ok := rhs.Items[0].(*ast.CallExpr); !ok {
		t.Fatalf("expected the first RHS item to be a CallExpr, got %#v", rhs.Items[0])
	}
}

func TestCommaSyntaxSwap(t *testing.T) {
	assign := commaTargets(t, mustParseOK(t, "x, y = [y, x]").Stmts[0], "x", "y")

	if _, ok := assign.Right.(*ast.ArrayExpr); !ok {
		t.Fatalf("expected the RHS to be an ArrayExpr, got %#v", assign.Right)
	}
}

// The original bug: a comma in a *nested* expression position belongs to the
// enclosing construct, so it must not be swallowed into a comma-target list.

func TestCommaInCallArgsIsNotDestructuring(t *testing.T) {
	// split(text, char) is a call with two args, not a call whose single arg
	// is a two-element comma list.
	prog := mustParseOK(t, "split(text, char)")

	call, ok := prog.Stmts[0].(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected a CallExpr, got %#v", prog.Stmts[0])
	}
	if len(call.Args) != 2 {
		t.Fatalf("expected 2 call args, got %d: %#v", len(call.Args), call.Args)
	}
	for i, name := range []string{"text", "char"} {
		if id, ok := call.Args[i].(*ast.Ident); !ok || id.Name != name {
			t.Fatalf("arg %d: expected Ident(%s), got %#v", i, name, call.Args[i])
		}
	}
}

func TestCommaInArrayLiteralIsNotDestructuring(t *testing.T) {
	// [a, b, c] is a three-element array, not a one-element array holding a
	// comma list.
	prog := mustParseOK(t, "[a, b, c]")

	arr, ok := prog.Stmts[0].(*ast.ArrayExpr)
	if !ok {
		t.Fatalf("expected an ArrayExpr, got %#v", prog.Stmts[0])
	}
	if len(arr.Items) != 3 {
		t.Fatalf("expected 3 items, got %d: %#v", len(arr.Items), arr.Items)
	}
}

func TestCommaSyntaxRejectsCompoundAssign(t *testing.T) {
	// Comma-syntax is only valid with plain '='; '+=' / '-=' must error.
	for _, src := range []string{"a, b += 1", "a, b -= 1"} {
		if prog := parseSrc(t, src); len(prog.Errors) == 0 {
			t.Errorf("%q: expected a parse error, got none", src)
		}
	}
}

func TestCommaSyntaxRejectsNonIdentifierTarget(t *testing.T) {
	// Every target in a comma list must be a plain identifier.
	for _, src := range []string{"a, b[0] = [1, 2]", "a, 3 = [1, 2]", "a, b.c = [1, 2]"} {
		if prog := parseSrc(t, src); len(prog.Errors) == 0 {
			t.Errorf("%q: expected a parse error, got none", src)
		}
	}
}

func TestMultiValueReturnWrapsInArray(t *testing.T) {
	// `return 1, 2` desugars to returning the array [1, 2], the mirror of the
	// `a, b = f()` destructuring on the caller's side.
	prog := mustParseOK(t, "\\() -> return 1, 2")

	fn := prog.Stmts[0].(*ast.FnExpr)
	ret, ok := fn.Body[0].(*ast.ReturnExpr)
	if !ok {
		t.Fatalf("expected a ReturnExpr as the fn body, got %#v", fn.Body[0])
	}

	arr, ok := ret.Value.(*ast.ArrayExpr)
	if !ok {
		t.Fatalf("expected the return value to be an ArrayExpr, got %#v", ret.Value)
	}
	if len(arr.Items) != 2 {
		t.Fatalf("expected 2 returned items, got %d: %#v", len(arr.Items), arr.Items)
	}
}

func TestSingleValueReturnIsNotWrapped(t *testing.T) {
	// A lone `return 42` must stay a scalar, not a one-element array.
	prog := mustParseOK(t, "\\() -> return 42")

	ret := prog.Stmts[0].(*ast.FnExpr).Body[0].(*ast.ReturnExpr)
	if _, ok := ret.Value.(*ast.ArrayExpr); ok {
		t.Fatalf("expected a scalar return value, got an ArrayExpr: %#v", ret.Value)
	}
	if lit, ok := ret.Value.(*ast.IntLit); !ok || lit.Value != 42 {
		t.Fatalf("expected the return value to be IntLit(42), got %#v", ret.Value)
	}
}
