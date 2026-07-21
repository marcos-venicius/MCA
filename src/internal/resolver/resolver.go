// Package resolver is a static pass that runs between parsing and evaluation.
// It walks the AST once and records, on every Ident, where the variable lives:
// how many scopes to climb (Depth) and which slot within that scope
// (FrameIndex).
//
// The scope structure here mirrors the interpreter exactly: a scope is pushed
// wherever the interpreter would push one (function bodies, if/while/for
// blocks) and a binding is declared wherever it would Define or Assign one.
// Keeping the two in lockstep is what makes the computed Depths line up with
// the Env chain the interpreter builds at runtime.
package resolver

import (
	"fmt"

	"mca/internal/ast"
)

// scope is one lexical frame during resolution. names maps a variable name to
// its slot; count is the next free slot.
type scope struct {
	names map[string]int
	count int
}

func (s *scope) declare(name string) int {
	if slot, ok := s.names[name]; ok {
		return slot
	}
	slot := s.count
	s.names[name] = slot
	s.count++
	return slot
}

// Resolver holds the scope stack; scopes[0] is the global scope.
type Resolver struct {
	scopes []*scope
}

// Resolve annotates every Ident in stmts with its Depth and FrameIndex.
func Resolve(stmts []ast.Expr) {
	r := &Resolver{}
	r.push()
	r.block(stmts)
	r.pop()
}

func (r *Resolver) push() {
	r.scopes = append(r.scopes, &scope{names: map[string]int{}})
}

func (r *Resolver) pop() {
	r.scopes = r.scopes[:len(r.scopes)-1]
}

func (r *Resolver) current() *scope {
	return r.scopes[len(r.scopes)-1]
}

// lookup walks the stack from the top, returning how far it had to climb.
func (r *Resolver) lookup(name string) (depth, slot int, ok bool) {
	for i := len(r.scopes) - 1; i >= 0; i-- {
		if slot, ok := r.scopes[i].names[name]; ok {
			return len(r.scopes) - 1 - i, slot, true
		}
	}
	return 0, 0, false
}

// read resolves a use of a variable. A name that isn't bound in any scope
// (a builtin, a forward reference, an import) gets Depth -1 so the runtime
// falls back to a by-name lookup.
func (r *Resolver) read(id *ast.Ident) {
	if depth, slot, ok := r.lookup(id.Name); ok {
		id.Depth, id.FrameIndex = depth, slot
		return
	}
	id.Depth, id.FrameIndex = -1, -1
}

// define binds a name unconditionally in the current scope, shadowing any
// outer one. Matches Env.Define/DefineConst: function parameters, loop
// variables, and const declarations.
func (r *Resolver) define(id *ast.Ident) {
	id.Depth = 0
	id.FrameIndex = r.current().declare(id.Name)
}

// assignTarget mirrors Env.Assign: a plain `x = ...` reuses an existing
// binding anywhere up the chain, and only declares a new local when the name
// isn't bound yet.
func (r *Resolver) assignTarget(id *ast.Ident) {
	if depth, slot, ok := r.lookup(id.Name); ok {
		id.Depth, id.FrameIndex = depth, slot
		return
	}
	id.Depth = 0
	id.FrameIndex = r.current().declare(id.Name)
}

func (r *Resolver) block(stmts []ast.Expr) {
	for _, stmt := range stmts {
		r.resolve(stmt)
	}
}

// scoped resolves a block in a fresh child scope, matching evalBlockNewScope.
func (r *Resolver) scoped(body []ast.Expr) {
	r.push()
	r.block(body)
	r.pop()
}

func (r *Resolver) resolve(e ast.Expr) {
	switch node := e.(type) {
	case nil:
		// The parser leaves nil in optional slots (a bare `while`, `break`,
		// `return`, or the missing bounds of a range-for). Nothing to resolve.

	case *ast.IntLit, *ast.FloatLit, *ast.BoolLit, *ast.StringLit, *ast.UnitLit:

	case *ast.Ident:
		r.read(node)

	case *ast.UnaryExpr:
		r.resolve(node.Operand)

	case *ast.BinaryExpr:
		r.resolve(node.Left)
		r.resolve(node.Right)

	case *ast.AssignExpr:
		r.resolveAssign(node)

	case *ast.CallExpr:
		r.resolve(node.Callee)
		for _, arg := range node.Args {
			r.resolve(arg)
		}

	case *ast.ArrayExpr:
		for _, item := range node.Items {
			r.resolve(item)
		}

	case *ast.MapExpr:
		for _, key := range node.Keys {
			// A bare identifier key (`{a: 1}`) is sugar for the string "a",
			// never a variable read -- only evaluated keys resolve.
			if _, isIdent := key.(*ast.Ident); !isIdent {
				r.resolve(key)
			}
		}
		for _, val := range node.Values {
			r.resolve(val)
		}

	case *ast.SquareExpr:
		r.resolve(node.Left)
		r.resolve(node.Index)

	case *ast.DotExpr:
		// Index is the field name used as a literal key, so only Left is a read.
		r.resolve(node.Left)

	case *ast.RangeExpression:
		r.resolve(node.Left)
		r.resolve(node.From)
		r.resolve(node.To)

	case *ast.FnExpr:
		r.push()
		for _, param := range node.Params {
			r.define(param)
		}
		r.block(node.Body)
		r.pop()

	case *ast.IfExpr:
		r.resolve(node.Condition)
		r.scoped(node.Then)
		for _, elif := range node.Elifs {
			// elif conditions run in the enclosing scope, before the branch.
			r.resolve(elif.Condition)
			r.scoped(elif.Body)
		}
		if node.Else != nil {
			r.scoped(node.Else)
		}

	case *ast.WhileExpr:
		r.resolve(node.Condition)
		r.scoped(node.Body)

	case *ast.ForRangeExpr:
		r.resolve(node.From)
		r.resolve(node.To)
		r.resolve(node.By)
		r.push()
		r.define(node.Index)
		r.block(node.Body)
		r.pop()

	case *ast.ForOfExpr:
		r.resolve(node.Target)
		r.push()
		r.define(node.Key)
		r.define(node.Value)
		r.block(node.Body)
		r.pop()

	case *ast.BreakExpr:
		r.resolve(node.Value)

	case *ast.ContinueExpr:
		// A bare `continue` carries no sub-expression -- nothing to resolve.

	case *ast.ReturnExpr:
		r.resolve(node.Value)

	default:
		panic(fmt.Sprintf("resolver: unhandled node type %T", e))
	}
}

func (r *Resolver) resolveAssign(e *ast.AssignExpr) {
	// The interpreter evaluates the right-hand side before binding the target
	// (see evalAssignRightSide), so resolve it first: a fresh local only
	// becomes visible from its declaration onward.
	r.resolve(e.Right)

	switch left := e.Left.(type) {
	case *ast.Ident:
		r.bindTarget(left, e.Const)

	case *ast.ArrayExpr:
		// destructuring: `a, b = pair`
		for _, item := range left.Items {
			if id, ok := item.(*ast.Ident); ok {
				r.bindTarget(id, e.Const)
			} else {
				r.resolve(item)
			}
		}

	case *ast.SquareExpr:
		// `arr[i] = v` -- the target already exists, nothing is declared.
		r.resolve(left.Left)
		r.resolve(left.Index)

	case *ast.DotExpr:
		// `m.k = v` -- Left is a read, Index is a literal key.
		r.resolve(left.Left)

	default:
		r.resolve(e.Left)
	}
}

func (r *Resolver) bindTarget(id *ast.Ident, isConst bool) {
	if isConst {
		r.define(id)
		return
	}
	r.assignTarget(id)
}
