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
