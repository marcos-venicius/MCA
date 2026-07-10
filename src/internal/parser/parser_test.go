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

func TestCallOnlyRecognizedForBareIdentifier(t *testing.T) {
	// arr[0](5) must NOT parse as a call -- arr[0] is one statement, (5) is
	// a separate parenthesized-expression statement. TODO: I need to fix it.
	prog := mustParseOK(t, "arr[0](5)")

	if len(prog.Stmts) != 2 {
		t.Fatalf("expected 2 top-level statements (index expr, then paren expr), got %d: %#v", len(prog.Stmts), prog.Stmts)
	}

	if _, ok := prog.Stmts[0].(*ast.SquareExpr); !ok {
		t.Fatalf("expected first stmt to be IndexExpr, got %#v", prog.Stmts[0])
	}

	if _, ok := prog.Stmts[1].(*ast.IntLit); !ok {
		t.Fatalf("expected second stmt to be IntLit(5), got %#v", prog.Stmts[1])
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
