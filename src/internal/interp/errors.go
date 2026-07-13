package interp

import (
	"fmt"
	"slices"

	"mca/internal/ast"
)

// RuntimeError is a "file:line:col: runtime error: msg" diagnostic. MCA has
// no try/catch construct (TODO: don't know yet how am I gonna handle errors),
// so the first runtime error anywhere is fatal to
// the whole program; this is implemented with panic/recover rather than
// os.Exit deep inside evaluation, so that
// a) the CLI can still print the message and exit(1) at the boundary, and
// b) tests and the import builtin's nested interpreter can recover it as a plain Go error instead of killing the whole test binary.
type RuntimeError struct {
	Filename  string
	Line, Col int
	Message   string
}

func (e *RuntimeError) Error() string {
	// TODO: I'm not sure how colors will work on another terminals
	if e.Filename != "" {
		return fmt.Sprintf("%s:%d:%d: \033[1;31mruntime error\033[0m: %s", e.Filename, e.Line, e.Col, e.Message)
	}
	return fmt.Sprintf("%d:%d: \033[1;31mruntime error\033[0m: %s", e.Line, e.Col, e.Message)
}

// throw panics with a *RuntimeError positioned at e's location; the nearest
// Run/eval boundary recovers it.
func throw(pos ast.Pos, format string, args ...any) {
	panic(&RuntimeError{
		Filename: pos.Filename,
		Line:     pos.Line,
		Col:      pos.Col,
		Message:  fmt.Sprintf(format, args...),
	})
}

// Throw is throw for native packages, which live outside this package and so
// cannot reach the unexported one. Position it at c.Site for a fault with the
// call as a whole, or c.At(i) to blame one argument.
func Throw(pos ast.Pos, format string, args ...any) {
	throw(pos, format, args...)
}

// expectKind raises a RuntimeError formatted as "unexpected data type.
// expected a 'X | Y' but got a 'Z'" if v's kind isn't one of the allowed
// kinds; returns v unchanged otherwise so call sites can chain it.
func expectKind(at ast.Expr, v Value, allowed ...Kind) Value {
	return expectKindAt(at.Pos(), v, allowed...)
}

// expectKindAt is expectKind for callers that hold a position rather than the
// expression it came from -- builtins, whose arguments arrive already
// evaluated and may not have any argument syntax behind them at all.
func expectKindAt(pos ast.Pos, v Value, allowed ...Kind) Value {
	if slices.Contains(allowed, v.Kind()) {
		return v
	}

	throw(pos, "unexpected data type. expected a '%s' but got a '%s'", KindsName(allowed...), v.Kind())
	panic("unreachable")
}
