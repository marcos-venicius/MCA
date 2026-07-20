package interp

import (
	"io"
	"math"
	"os"

	"mca/internal/ast"
	"mca/internal/resolver"
)

// ControlFlow tags how an evaluation completed: falling through normally,
// or unwinding because of a break or return (later add `continue`).
type ControlFlow int

const (
	FlowNormal ControlFlow = iota
	FlowBreak
	FlowReturn
)

// EvalResult is what every Eval call produces: a value plus how it got
// there. Only block-statement sequencing (evalBlock) inspects Flow to decide
// whether to keep running; everywhere else that consumes a sub-expression's
// result (binary operands, call arguments, array items, ...) uses only
// .Value and discards .Flow, since a bare break/return can only meaningfully
// unwind through statement sequencing, not through an arbitrary operand
// position.
type EvalResult struct {
	Value Value
	Flow  ControlFlow
}

func normal(v Value) EvalResult { return EvalResult{Value: v, Flow: FlowNormal} }

// BuiltinFn is a builtin's implementation. Builtins receive their arguments
// already evaluated, exactly like a user function does -- which is what makes
// a builtin storable in a variable and passable to map/filter/sort. They took
// raw AST nodes and evaluated them by hand once, back when a builtin could
// only ever be *called*; nothing needed that laziness, and the positions the
// error messages actually wanted are carried by Call instead.
type BuiltinFn func(in *Interp, c *Call) Value

// Call is a single builtin invocation: the argument values, plus the source
// positions its diagnostics point at.
type Call struct {
	Name string  // the builtin's registered name, for error messages
	Site ast.Pos // where the call itself appears

	Args   []Value
	argPos []ast.Pos // per-argument positions; empty when there is no arg syntax
}

// At is the position to blame for argument i. A builtin reached indirectly
// (map(arr, upper), or any other call through a value) has no argument syntax
// to point at, so its diagnostics collapse onto the call site.
func (c *Call) At(i int) ast.Pos {
	if i < len(c.argPos) {
		return c.argPos[i]
	}
	return c.Site
}

// N is the number of arguments passed. Only variadic builtins need it; the
// shared call path has already enforced the count for every other one.
func (c *Call) N() int { return len(c.Args) }

// StringArg is argument i, required to be a string: the same
// expect-then-unwrap the builtins in this package spell out by hand, exported
// so a native package (internal/packages/...) can validate its arguments
// without reaching for interp's unexported helpers. Raises the usual
// "unexpected data type" runtime error, blamed on argument i, if it is not a
// string.
func (c *Call) StringArg(i int) string {
	return stringOf(expectKindAt(c.At(i), c.Args[i], KString))
}

// IntArg is argument i, required to be an int -- see StringArg.
func (c *Call) IntArg(i int) int64 {
	return intOf(expectKindAt(c.At(i), c.Args[i], KInt))
}

// ArrayArg is argument i, required to be an array -- see StringArg.
func (c *Call) ArrayArg(i int) *Array {
	return arrayOf(expectKindAt(c.At(i), c.Args[i], KArray))
}

// Arg is argument i, required to be one of the allowed kinds: the general
// form of StringArg/IntArg for a native package whose argument may span
// several kinds. Returns the value still wrapped, so the caller type-switches
// on it; raises the usual "unexpected data type" runtime error, blamed on
// argument i, otherwise.
func (c *Call) Arg(i int, allowed ...Kind) Value {
	return expectKindAt(c.At(i), c.Args[i], allowed...)
}

// Interp is one running MCA program. Out/Err are settable so tests (and the
// CLI) can redirect them. Args holds the language-level argv, with Args[0]
// conventionally the script path.
type Interp struct {
	Global  *Env
	Current *Env

	Out  io.Writer
	Err  io.Writer
	Args []string
}

func New() *Interp {
	// Builtins are ordinary constant values, not a side table consulted at
	// call time -- that is what makes them first-class: `sort` is a value you
	// can pass around, not just a name you may write before a '('.
	//
	// They get a frame of their own, one below the global scope, so that a
	// program can still bind `year` or `help` as a variable (an assignment
	// shadows the builtin in the assigning scope) without any program
	// anywhere being able to *overwrite* what `sort` means.
	b := NewBuiltinEnv()
	slot := 0
	for name, n := range builtins {
		b.define(slot, name, FnValV(&FnValue{Native: n}), true)
		slot++
	}

	g := NewEnv(b)

	return &Interp{Global: g, Current: g, Out: os.Stdout, Err: os.Stderr}
}

// Run evaluates stmts under the single recover boundary for the whole
// program: a runtime error anywhere, including inside an imported module, is
// fatal to the entire run. Callers must ensure the program parsed with zero
// errors first.
func (in *Interp) Run(stmts []ast.Expr) (result Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(*RuntimeError); ok {
				err = re
				return
			}
			panic(r)
		}
	}()

	return in.runStatements(stmts), nil
}

// runStatements has no recover boundary of its own -- used directly (not via
// Run) by the import() builtin, so a runtime error inside an imported
// module's top-level code panics straight through to whichever Run call is
// outermost, keeping "a runtime error anywhere kills the whole program" true
// even across module boundaries.
func (in *Interp) runStatements(stmts []ast.Expr) Value {
	// Resolve here rather than at the call sites: this is the one place both a
	// top-level program (via Run) and an imported module reach before their
	// statements are evaluated.
	resolver.Resolve(stmts)

	result := UnitV()

	for _, stmt := range stmts {
		r := in.Eval(stmt)

		if r.Flow == FlowBreak {
			throw(stmt.Pos(), "cannot use 'break' outside of a loop")
		}
		if r.Flow == FlowReturn {
			throw(stmt.Pos(), "cannot use 'return' outside of a function")
		}

		result = r.Value
	}

	return result
}

// Eval dispatches on the concrete type of e. A central type switch (rather
// than an Eval method per AST node) is the idiom Go's own stdlib uses for
// this same shape of problem (e.g. text/template's state.walk) -- it also
// sidesteps a real import cycle here, since ast can't depend on interp for
// an Eval method's return type while interp already depends on ast for the
// node types themselves.
func (in *Interp) Eval(e ast.Expr) EvalResult {
	switch node := e.(type) {
	case *ast.StringLit:
		return normal(StringV(node.Value))
	case *ast.UnitLit:
		return normal(UnitV())
	case *ast.BoolLit:
		return normal(BoolV(node.Value))
	case *ast.IntLit:
		return normal(IntV(node.Value))
	case *ast.FloatLit:
		return normal(FloatV(node.Value))
	case *ast.Ident:
		v, ok := in.lookupVar(node)
		if !ok {
			throw(node.Pos(), "variable '%s' does not exist", node.Name)
		}
		return normal(v)
	case *ast.AssignExpr:
		return in.evalAssign(node)
	case *ast.UnaryExpr:
		return in.evalUnary(node)
	case *ast.BinaryExpr:
		return in.evalBinary(node)
	case *ast.CallExpr:
		return in.evalCall(node)
	case *ast.FnExpr:
		return normal(FnValV(&FnValue{Node: node, Env: in.Current}))
	case *ast.IfExpr:
		return in.evalIf(node)
	case *ast.WhileExpr:
		return in.evalWhile(node)
	case *ast.ForRangeExpr:
		return in.evalForRange(node)
	case *ast.BreakExpr:
		return in.evalBreak(node)
	case *ast.ReturnExpr:
		return in.evalReturn(node)
	case *ast.ArrayExpr:
		return in.evalArrayLit(node)
	case *ast.MapExpr:
		return in.evalMapLit(node)
	case *ast.SquareExpr:
		return in.evalSquare(node)
	case *ast.DotExpr:
		return in.evalDot(node)
	case *ast.RangeExpression:
		return in.evalRangeExpression(node)
	case *ast.ForOfExpr:
		return in.evalForOf(node)
	default:
		throw(e.Pos(), "internal: expression kind %T not yet implemented", e)
		panic("unreachable")
	}
}

// lookupVar reads a variable through its resolved slot, falling back to a
// by-name lookup for the cases the resolver marks Depth == -1 (builtins,
// forward references). The slot itself can also be empty if the read runs
// before the assignment that fills it, so an empty slot falls back too.
func (in *Interp) lookupVar(id *ast.Ident) (Value, bool) {
	if id.Depth >= 0 {
		if b := in.Current.bySlot(id.Depth, id.FrameIndex); b != nil {
			return b.value, true
		}
	}
	if b, ok := in.Current.byName(id.Name); ok {
		return b.value, true
	}
	return Value{}, false
}

func (in *Interp) evalBlock(body []ast.Expr) EvalResult {
	result := normal(UnitV())

	for _, stmt := range body {
		result = in.Eval(stmt)
		if result.Flow != FlowNormal {
			return result
		}
	}

	return result
}

// pushScope/popScope enter and leave a lexical scope. There's no manual free
// the way C needed -- Go's GC reclaims the scope once nothing (including an
// escaping closure) still references it.
func (in *Interp) pushScope() (parent *Env) {
	parent = in.Current
	in.Current = NewEnv(parent)
	return parent
}

func (in *Interp) popScope(parent *Env) {
	in.Current = parent
}

func (in *Interp) evalBlockNewScope(body []ast.Expr) EvalResult {
	parent := in.pushScope()
	defer in.popScope(parent)
	return in.evalBlock(body)
}

// ---- unary ----

func (in *Interp) evalUnary(e *ast.UnaryExpr) EvalResult {
	res := in.Eval(e.Operand)

	switch e.Op {
	case ast.MinusOp:
		var out Value
		switch res.Value.Kind() {
		case KInt:
			out = IntV(-intOf(res.Value))
		case KFloat:
			out = FloatV(-floatOf(res.Value))
		default:
			throw(e.Pos(), "unexpected data type. expected a '%s' but got a '%s'", KindsName(KInt, KFloat), res.Value.Kind())
		}
		return EvalResult{Value: out, Flow: res.Flow}

	case ast.NotOp:
		isTrue, _ := Truthy(res.Value)

		return EvalResult{Value: BoolV(!isTrue), Flow: res.Flow}

	case ast.FactorialOp:
		v := expectKind(e, res.Value, KInt, KFloat)
		return EvalResult{Value: calculateFactorial(v), Flow: res.Flow}

	case ast.BitNotOp:
		// int-only, like the binary bitwise operators.
		v := intOf(expectKind(e.Operand, res.Value, KInt))
		return EvalResult{Value: IntV(^v), Flow: res.Flow}
	}

	panic("evalUnary: unhandled operator")
}

// calculateFactorial computes tgamma(x+1); negative operands (int, or float
// with an exact integral value) yield float NaN instead, since factorial is
// undefined there.
func calculateFactorial(v Value) Value {
	var val float64
	isNegativeInt := false
	isInt := false

	switch v.Kind() {
	case KInt:
		i := intOf(v)
		val = float64(i)
		isNegativeInt = i < 0
		isInt = true
	case KFloat:
		val = floatOf(v)
		isNegativeInt = val < 0 && val == float64(int64(val))
	}

	if isNegativeInt {
		return FloatV(math.NaN())
	}

	x := math.Gamma(val + 1.0)

	if isInt {
		return IntV(int64(x))
	}
	return FloatV(x)
}

// ---- binary ----

func isComparisonOp(op ast.BinaryOp) bool {
	switch op {
	case ast.EqualOp, ast.NotEqualOp, ast.GtOp, ast.LtOp, ast.GteOp, ast.LteOp:
		return true
	}
	return false
}

// asFloat widens an already kind-checked int/float/bool value to float64.
func asFloat(v Value) float64 {
	switch v.Kind() {
	case KFloat:
		return floatOf(v)
	case KBool:
		if boolOf(v) {
			return 1
		}
		return 0
	default: // KInt
		return float64(intOf(v))
	}
}

// asIntLike reads an already kind-checked int/bool value as int64.
func asIntLike(v Value) int64 {
	if v.Kind() == KInt {
		return intOf(v)
	}
	if boolOf(v) {
		return 1
	}
	return 0
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func binaryOpOnFloats(op ast.BinaryOp, l, r float64) float64 {
	switch op {
	case ast.PlusOp:
		return l + r
	case ast.TimesOp:
		return l * r
	case ast.DivideOp:
		return l / r
	case ast.SubtractOp:
		return l - r
	case ast.ModOp:
		return math.Mod(l, r)
	case ast.PowOp:
		return math.Pow(l, r)
	case ast.EqualOp:
		return boolToFloat(l == r)
	case ast.NotEqualOp:
		return boolToFloat(l != r)
	case ast.GtOp:
		return boolToFloat(l > r)
	case ast.LtOp:
		return boolToFloat(l < r)
	case ast.GteOp:
		return boolToFloat(l >= r)
	case ast.LteOp:
		return boolToFloat(l <= r)
	}

	panic("binaryOpOnFloats: unhandled operator")
}

// isNumericKind reports whether k is one of the kinds that == and !=
// coerce together (int/float/bool), matching the coercion binaryOpOnFloats
// and asIntLike already do for the other arithmetic/relational operators.
func isNumericKind(k Kind) bool {
	return k == KInt || k == KFloat || k == KBool
}

func compareTwoValues(a, b Value) bool {
	if isNumericKind(a.Kind()) && isNumericKind(b.Kind()) {
		if a.Kind() == KFloat || b.Kind() == KFloat {
			return asFloat(a) == asFloat(b)
		}
		return asIntLike(a) == asIntLike(b)
	}

	if a.Kind() != b.Kind() {
		return false
	}

	switch a.Kind() {
	case KUnit: // both are equal because unit doesn't have a value
		return true
	case KString:
		return stringOf(b) == stringOf(a)
	case KArray:
		return compareTwoArrays(arrayOf(a), arrayOf(b))
	case KMap:
		// TODO: we don't do cycle checks. if we have a cyclic reference, it's gonna overflow the stack and panic
		return compareTwoMaps(mapOf(a), mapOf(b))
	case KFn:
		// pointer equality
		return fnOf(a) == fnOf(b)
	}

	return false
}

func compareTwoArrays(a, b *Array) bool {
	// same pointer (same value)
	if a == b {
		return true
	}

	if len(a.Items) != len(b.Items) {
		return false
	}

	for i := range len(a.Items) {
		v1 := a.Items[i]
		v2 := b.Items[i]

		if !compareTwoValues(v1, v2) {
			return false
		}
	}

	return true
}

func compareTwoMaps(a, b *Map) bool {
	// same pointer (same value)
	if a == b {
		return true
	}

	if a.Len() != b.Len() {
		return false
	}

	for k1, v1 := range a.values {
		v2, ok := b.values[k1]

		if !ok {
			return false
		}

		if !compareTwoValues(v1, v2) {
			return false
		}
	}

	return true
}

func (in *Interp) evalBinary(e *ast.BinaryExpr) EvalResult {
	switch e.Op {
	case ast.AndOp:
		leftRes := in.Eval(e.Left)

		if t, _ := Truthy(leftRes.Value); !t {
			return normal(BoolV(false))
		}

		rightRes := in.Eval(e.Right)

		if t, _ := Truthy(rightRes.Value); !t {
			return normal(BoolV(false))
		}

		return normal(BoolV(true))

	case ast.OrOp:
		leftRes := in.Eval(e.Left)

		if t, _ := Truthy(leftRes.Value); t {
			return normal(BoolV(true))
		}

		rightRes := in.Eval(e.Right)

		if t, _ := Truthy(rightRes.Value); t {
			return normal(BoolV(true))
		}

		return normal(BoolV(false))
	}

	left := in.Eval(e.Left).Value

	if e.Op == ast.EqualOp {
		right := in.Eval(e.Right).Value
		return normal(BoolV(compareTwoValues(left, right)))
	}

	if e.Op == ast.NotEqualOp {
		right := in.Eval(e.Right).Value
		return normal(BoolV(!compareTwoValues(left, right)))
	}

	// The bitwise operators are int-only -- no float/bool coercion, since a
	// bit pattern only makes sense on an int. Same left-before-right check
	// order as below.
	switch e.Op {
	case ast.ShlOp, ast.ShrOp, ast.XorOp, ast.BitAndOp, ast.BitOrOp:
		l := intOf(expectKind(e.Left, left, KInt))
		r := intOf(expectKind(e.Right, in.Eval(e.Right).Value, KInt))

		switch e.Op {
		case ast.ShlOp:
			if r < 0 {
				throw(e.Right.Pos(), "negative shift count %d", r)
			}
			return normal(IntV(l << uint64(r)))
		case ast.ShrOp:
			if r < 0 {
				throw(e.Right.Pos(), "negative shift count %d", r)
			}
			// '>>' is an arithmetic shift: the sign bit fills in from the
			// left, so a negative left operand stays negative.
			return normal(IntV(l >> uint64(r)))
		case ast.XorOp:
			return normal(IntV(l ^ r))
		case ast.BitAndOp:
			return normal(IntV(l & r))
		default: // ast.BitOrOp
			return normal(IntV(l | r))
		}
	}

	// Unlike ==/!=, the arithmetic/relational operators below only accept
	// int/float/bool -- type-check the left operand before evaluating the
	// right one, so a bad left type is reported without evaluating (and
	// side-effecting on) the right side at all.
	left = expectKind(e.Left, left, KInt, KFloat, KBool)
	right := expectKind(e.Right, in.Eval(e.Right).Value, KInt, KFloat, KBool)

	returnsBool := isComparisonOp(e.Op)

	if left.Kind() == KFloat || right.Kind() == KFloat {
		result := binaryOpOnFloats(e.Op, asFloat(left), asFloat(right))

		if returnsBool {
			return normal(BoolV(result != 0.0))
		}
		return normal(FloatV(result))
	}

	l := asIntLike(left)
	r := asIntLike(right)

	if e.Op == ast.ModOp && r == 0 {
		// Native int %/0 has no well-defined value to preserve here (it's
		// undefined behavior, typically SIGFPE, in C); raise a clean runtime
		// error instead of letting Go's own divide-by-zero panic escape
		// uncontrolled.
		throw(e.Pos(), "division by zero")
	}

	var result float64

	switch e.Op {
	case ast.PlusOp:
		result = float64(l + r)
	case ast.TimesOp:
		result = float64(l * r)
	case ast.DivideOp:
		result = float64(l) / float64(r) // always true division
	case ast.SubtractOp:
		result = float64(l - r)
	case ast.ModOp:
		result = float64(l % r)
	case ast.PowOp:
		result = math.Pow(float64(l), float64(r))
	case ast.EqualOp:
		result = boolToFloat(l == r)
	case ast.NotEqualOp:
		result = boolToFloat(l != r)
	case ast.GtOp:
		result = boolToFloat(l > r)
	case ast.LtOp:
		result = boolToFloat(l < r)
	case ast.GteOp:
		result = boolToFloat(l >= r)
	case ast.LteOp:
		result = boolToFloat(l <= r)
	}

	if returnsBool {
		return normal(BoolV(result != 0.0))
	}

	// TODO: I'm still not sure if that's the best way of handling it
	if math.Mod(result, 1.0) != 0.0 {
		return normal(FloatV(result))
	}

	return normal(IntV(int64(result)))
}

// ---- assignment ----

func (in *Interp) evalAssign(e *ast.AssignExpr) EvalResult {
	rightRes := in.evalAssignRightSide(e)

	switch left := e.Left.(type) {
	case *ast.ArrayExpr:
		if rightRes.Value.Kind() != KArray {
			throw(e.Right.Pos(), "you cannot use comma-syntax with non array values")
		}

		value := arrayOf(rightRes.Value)

		if len(left.Items) != len(value.Items) {
			throw(e.Left.Pos(), "expected to have %d items but got %d", len(left.Items), len(value.Items))
		}

		for i, expr := range left.Items {
			ident := expr.(*ast.Ident)

			if e.Const {
				in.declareConst(ident, value.Items[i])
			} else if !in.Current.assign(ident.Depth, ident.FrameIndex, ident.Name, value.Items[i]) {
				throw(ident.Pos(), "you cannot modify constant values. '%s' is a constant", ident.Name)
			}
		}

		return rightRes

	case *ast.Ident:
		if e.Const {
			in.declareConst(left, rightRes.Value)
		} else if !in.Current.assign(left.Depth, left.FrameIndex, left.Name, rightRes.Value) {
			throw(left.Pos(), "you cannot modify constant values. '%s' is a constant", left.Name)
		}
		return rightRes

	case *ast.SquareExpr:
		in.storeSquareAssign(e, left, rightRes.Value)
		return rightRes

	case *ast.DotExpr:
		in.storeDotAssign(e, left, rightRes.Value)
		return rightRes
	}

	return rightRes
}

// declareConst introduces `const name = value` in the current scope. It
// refuses to overwrite a name this scope already owns -- a constant you can
// redeclare next to itself isn't one -- but says nothing about outer scopes,
// so an inner scope may still shadow a constant (including a builtin) with a
// binding of its own.
func (in *Interp) declareConst(name *ast.Ident, v Value) {
	if in.Current.hasLocal(name.Name) {
		throw(name.Pos(), "'%s' is already defined in this scope, so it cannot be declared as a constant", name.Name)
	}

	in.Current.define(name.FrameIndex, name.Name, v, true)
}

// evalAssignRightSide computes the value (and, for plain `=` only, the
// control flow) an assignment expression evaluates to. For plain `=` the
// right-hand side's EvalResult is returned completely unmodified -- unlike
// almost every other multi-operand construct here (binary operands, call
// arguments, array items, ...), which keep only .Value and drop .Flow. That
// asymmetry is what lets `x = if cond { break v }` propagate the break out
// through the assignment and into the enclosing loop. For `+=`/`-=` the
// result is always Flow-Normal (freshly constructed), regardless of what
// flow evaluating the left/right sub-expressions produced.
// TODO: work better on this later, I'm alreay tired.
func (in *Interp) evalAssignRightSide(e *ast.AssignExpr) EvalResult {
	rightRes := in.Eval(e.Right)

	if e.Op == ast.Assign {
		return rightRes
	}

	leftRes := in.Eval(e.Left)
	left := expectKind(e.Left, leftRes.Value, KInt, KFloat)
	right := expectKind(e.Right, rightRes.Value, KInt, KFloat)

	if left.Kind() == KFloat || right.Kind() == KFloat {
		lv, rv := asFloat(left), asFloat(right)
		if e.Op == ast.AddAssign {
			return normal(FloatV(lv + rv))
		}
		return normal(FloatV(lv - rv))
	}

	li, ri := intOf(left), intOf(right)
	if e.Op == ast.AddAssign {
		return normal(IntV(li + ri))
	}
	return normal(IntV(li - ri))
}

// ---- if/while/for-range/break/return ----

func (in *Interp) evalIf(e *ast.IfExpr) EvalResult {
	condRes := in.Eval(e.Condition)
	t, ok := Truthy(condRes.Value)
	if !ok {
		throw(e.Condition.Pos(), "failed to check truthiness of '%s' data type on that 'if'", condRes.Value.Kind())
	}

	if t {
		return in.evalBlockNewScope(e.Then)
	}

	for _, elif := range e.Elifs {
		er := in.Eval(elif.Condition)
		et, ok := Truthy(er.Value)
		if !ok {
			throw(elif.Condition.Pos(), "failed to check truthiness of '%s' data type on that 'elif'", er.Value.Kind())
		}

		if et {
			return in.evalBlockNewScope(elif.Body)
		}
	}

	if e.Else != nil {
		return in.evalBlockNewScope(e.Else)
	}

	return normal(UnitV())
}

func (in *Interp) evalWhile(e *ast.WhileExpr) EvalResult {
	last := normal(UnitV())

	for {
		if e.Condition != nil {
			condRes := in.Eval(e.Condition)
			t, ok := Truthy(condRes.Value)
			if !ok {
				throw(e.Condition.Pos(), "failed to check truthiness of '%s' data type on that 'loop'", condRes.Value.Kind())
			}
			if !t {
				break
			}
		}

		if e.Body != nil {
			last = in.evalBlockNewScope(e.Body)

			if last.Flow == FlowReturn {
				return last
			}
			if last.Flow == FlowBreak {
				last = normal(last.Value)
				break
			}
		}
	}

	return last
}

func (in *Interp) evalForRange(e *ast.ForRangeExpr) EvalResult {
	fromRes := in.Eval(e.From)
	fromV := expectKind(e.From, fromRes.Value, KInt)

	if e.To == nil {
		return in.runForRange(e, 0, intOf(fromV), 1)
	}

	toRes := in.Eval(e.To)
	toV := expectKind(e.To, toRes.Value, KInt)

	if e.By == nil {
		return in.runForRange(e, intOf(fromV), intOf(toV), 1)
	}

	byRes := in.Eval(e.By)
	byV := expectKind(e.By, byRes.Value, KInt)

	return in.runForRange(e, intOf(fromV), intOf(toV), intOf(byV))
}

// runForRange drives all three `for i : ...` shapes. Direction (ascending vs
// descending) is decided by whether from*to crosses zero -- a quirk pinned
// by an existing test and examples/loops.mca, so it's replicated exactly
// rather than "fixed" to something more obvious like sign(by). break here
// stops the loop and yields its value, same as while.
// TODO: think more about this later.
func (in *Interp) runForRange(e *ast.ForRangeExpr, from, to, by int64) EvalResult {
	last := normal(UnitV())

	isNegative := from*to < 0

	for i := from; forRangeCond(isNegative, i, to); i += by {
		if e.Body == nil {
			continue
		}

		parent := in.pushScope()
		in.Current.define(e.Index.FrameIndex, e.Index.Name, IntV(i), false)
		last = in.evalBlock(e.Body)
		in.popScope(parent)

		if last.Flow == FlowReturn {
			return last
		}
		if last.Flow == FlowBreak {
			last = normal(last.Value)
			break
		}
	}

	return last
}

func forRangeCond(isNegative bool, i, to int64) bool {
	if isNegative {
		return i > to
	}
	return i < to
}

func (in *Interp) evalBreak(e *ast.BreakExpr) EvalResult {
	if e.Value != nil {
		r := in.Eval(e.Value)
		return EvalResult{Value: r.Value, Flow: FlowBreak}
	}
	return EvalResult{Value: UnitV(), Flow: FlowBreak}
}

func (in *Interp) evalReturn(e *ast.ReturnExpr) EvalResult {
	if e.Value != nil {
		r := in.Eval(e.Value)
		return EvalResult{Value: r.Value, Flow: FlowReturn}
	}
	return EvalResult{Value: UnitV(), Flow: FlowReturn}
}

// ---- calls ----

// calleeLabel produces a short, human-readable name for a call's callee, for
// arity-mismatch error messages -- just the bare/field name for `f(...)` and
// `m.f(...)` calls, a generic placeholder for every other callee shape (e.g.
// `arr[0](...)`, `(\() -> 1)()`).
func calleeLabel(e ast.Expr) string {
	switch node := e.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.DotExpr:
		if ident, ok := node.Index.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return "<anonymous function>"
}

// evalCall has no special case for builtins: a builtin is just a global
// constant holding an FnValue, so a bare `sort(...)` callee resolves through
// the ordinary identifier lookup below, exactly like `m.f(...)`, `arr[0](...)`
// or `(\() -> 1)()` always did.
// fnLabel names a callable that a builtin is about to invoke on the user's
// behalf (map/filter/sort applying their callback). There is no callee syntax
// to read a name from in that position, so a builtin passed as the callback
// answers with its own registered name and anything else stays anonymous.
func fnLabel(fv *FnValue) string {
	if fv.Native != nil {
		return fv.Native.Name
	}
	return "<anonymous function>"
}

func (in *Interp) evalCall(e *ast.CallExpr) EvalResult {
	calleeVal := in.Eval(e.Callee).Value

	if calleeVal.Kind() != KFn {
		throw(e.Pos(), "you are trying to call a '%s' value, which is not a function", calleeVal.Kind())
	}
	fv := fnOf(calleeVal)

	return normal(in.callFn(fv, e.Pos(), calleeLabel(e.Callee), e.Args))
}

// callFn is a call written in source: the arguments are evaluated left to
// right in the caller's (current) environment, and their positions are kept
// so a builtin can still blame the exact argument that was wrong.
func (in *Interp) callFn(fv *FnValue, callPos ast.Pos, name string, argExprs []ast.Expr) Value {
	args := make([]Value, len(argExprs))
	argPos := make([]ast.Pos, len(argExprs))

	for i, argExpr := range argExprs {
		args[i] = in.Eval(argExpr).Value
		argPos[i] = argExpr.Pos()
	}

	return in.call(fv, callPos, name, args, argPos)
}

// callFnValue is a call made from Go with values already in hand (map/filter/
// sort applying their callback). There is no argument syntax to point at, so
// argPos is empty and any diagnostic falls back to the call site.
func (in *Interp) callFnValue(fv *FnValue, callPos ast.Pos, name string, args []Value) Value {
	return in.call(fv, callPos, name, args, nil)
}

// call is the one place a callable is entered, builtin or not: check the
// argument count, then either hand the values to the native implementation or
// bind them into a fresh call frame.
//
// That frame's parent is the function's *captured* environment (fv.Env, not
// the caller's environment) -- this is what makes MCA's functions
// lexically-scoped closures rather than dynamically-scoped ones.
func (in *Interp) call(fv *FnValue, callPos ast.Pos, name string, args []Value, argPos []ast.Pos) Value {
	if n := fv.Arity(); n >= 0 {
		if len(args) > n {
			throw(callPos, "too many arguments %s(...). expected %d but got %d", name, n, len(args))
		} else if len(args) < n {
			throw(callPos, "too few arguments %s(...). expected %d but got %d", name, n, len(args))
		}
	}

	if fv.Native != nil {
		return fv.Native.Fn(in, &Call{
			Name:   fv.Native.Name,
			Site:   callPos,
			Args:   args,
			argPos: argPos,
		})
	}

	fnEnv := NewEnv(fv.Env)
	for i, param := range fv.Node.Params {
		fnEnv.define(param.FrameIndex, param.Name, args[i], false)
	}

	callerEnv := in.Current
	in.Current = fnEnv
	result := in.evalBlock(fv.Node.Body)
	in.Current = callerEnv

	return result.Value
}
