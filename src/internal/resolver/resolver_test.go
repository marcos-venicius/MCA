package resolver

import (
	"fmt"
	"testing"

	"mca/internal/ast"
	"mca/internal/lexer"
	"mca/internal/parser"
)

// collectIdents returns every Ident in source order, including the dot/map
// keys the resolver deliberately leaves alone -- so a test can assert those
// stay untouched.
func collectIdents(t *testing.T, stmts []ast.Expr) []*ast.Ident {
	t.Helper()

	var out []*ast.Ident
	var walk func(ast.Expr)
	walk = func(e ast.Expr) {
		switch n := e.(type) {
		case nil:
		case *ast.IntLit, *ast.FloatLit, *ast.BoolLit, *ast.StringLit, *ast.UnitLit:
		case *ast.Ident:
			out = append(out, n)
		case *ast.UnaryExpr:
			walk(n.Operand)
		case *ast.BinaryExpr:
			walk(n.Left)
			walk(n.Right)
		case *ast.AssignExpr:
			walk(n.Left)
			walk(n.Right)
		case *ast.CallExpr:
			walk(n.Callee)
			for _, a := range n.Args {
				walk(a)
			}
		case *ast.ArrayExpr:
			for _, it := range n.Items {
				walk(it)
			}
		case *ast.MapExpr:
			for _, k := range n.Keys {
				walk(k)
			}
			for _, v := range n.Values {
				walk(v)
			}
		case *ast.SquareExpr:
			walk(n.Left)
			walk(n.Index)
		case *ast.DotExpr:
			walk(n.Left)
			walk(n.Index)
		case *ast.FnExpr:
			for _, p := range n.Params {
				walk(p)
			}
			for _, s := range n.Body {
				walk(s)
			}
		case *ast.IfExpr:
			walk(n.Condition)
			for _, s := range n.Then {
				walk(s)
			}
			for _, el := range n.Elifs {
				walk(el.Condition)
				for _, s := range el.Body {
					walk(s)
				}
			}
			for _, s := range n.Else {
				walk(s)
			}
		case *ast.WhileExpr:
			walk(n.Condition)
			for _, s := range n.Body {
				walk(s)
			}
		case *ast.ForRangeExpr:
			walk(n.From)
			walk(n.To)
			walk(n.By)
			walk(n.Index)
			for _, s := range n.Body {
				walk(s)
			}
		case *ast.ForOfExpr:
			walk(n.Target)
			walk(n.Key)
			walk(n.Value)
			for _, s := range n.Body {
				walk(s)
			}
		case *ast.BreakExpr:
			walk(n.Value)
		case *ast.ReturnExpr:
			walk(n.Value)
		default:
			t.Fatalf("collectIdents: unhandled node %T", e)
		}
	}

	for _, s := range stmts {
		walk(s)
	}
	return out
}

func mustResolve(t *testing.T, src string) []*ast.Ident {
	t.Helper()

	l := lexer.New("test.mca", src)
	toks := l.Tokenize()
	if len(l.Errors) > 0 {
		t.Fatalf("lex errors for %q: %v", src, l.Errors)
	}

	prog := parser.Parse("test.mca", toks)
	if len(prog.Errors) > 0 {
		t.Fatalf("parse errors for %q: %v", src, prog.Errors)
	}

	Resolve(prog.Stmts)
	return collectIdents(t, prog.Stmts)
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string // "name:depth:frameindex" per ident, source order
	}{
		{
			name: "reassignment keeps the same slot",
			src:  "x = 1; x = 2",
			want: []string{"x:0:0", "x:0:0"},
		},
		{
			name: "read resolves to the declaring slot",
			src:  "x = 1; y = x",
			want: []string{"x:0:0", "y:0:1", "x:0:0"},
		},
		{
			name: "closure reads an outer variable one scope up",
			src:  "x = 1; f = \\() -> x",
			want: []string{"x:0:0", "f:0:1", "x:1:0"},
		},
		{
			name: "self-reference in an initializer stays unresolved",
			// f is bound only after its initializer is resolved, so the body's
			// f falls back to a by-name lookup (depth -1).
			src:  "f = \\() -> f()",
			want: []string{"f:0:0", "f:-1:-1"},
		},
		{
			name: "a parameter shadows an outer variable",
			src:  "x = 1; f = \\(x) -> x",
			want: []string{"x:0:0", "f:0:1", "x:0:0", "x:0:0"},
		},
		{
			name: "range loop variable owns a slot in the loop scope",
			src:  "n = 5; for i : n { i }",
			want: []string{"n:0:0", "n:0:0", "i:0:0", "i:0:0"},
		},
		{
			name: "for-of binds key and value in the loop scope",
			src:  "m = {}; for k, v : m { k }",
			want: []string{"m:0:0", "m:0:0", "k:0:0", "v:0:1", "k:0:0"},
		},
		{
			name: "destructuring declares each target",
			src:  "arr = [1, 2]; a, b = arr",
			want: []string{"arr:0:0", "a:0:1", "b:0:2", "arr:0:0"},
		},
		{
			name: "const declares and later reads resolve to it",
			src:  "const c = 1; c",
			want: []string{"c:0:0", "c:0:0"},
		},
		{
			name: "block scope nests one level deeper",
			src:  "x = 1; if x { y = 2; x }",
			want: []string{"x:0:0", "x:0:0", "y:0:0", "x:1:0"},
		},
		{
			name: "builtins do not resolve to a slot",
			src:  "len([1])",
			want: []string{"len:-1:-1"},
		},
		{
			name: "dot field name is a key, not a variable",
			src:  "m = {}; m.field",
			want: []string{"m:0:0", "m:0:0", "field:0:0"},
		},
		{
			name: "map literal identifier key is not a variable",
			src:  "x = {field: 1}",
			want: []string{"x:0:0", "field:0:0"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idents := mustResolve(t, tc.src)

			got := make([]string, len(idents))
			for i, id := range idents {
				got[i] = fmt.Sprintf("%s:%d:%d", id.Name, id.Depth, id.FrameIndex)
			}

			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("ident %d: got %s, want %s (all: %v)", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}
