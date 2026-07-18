package ast

// TODO: I need to centralize this struct, it's repeated in so many places already
type Pos struct {
	Filename  string
	Line, Col int
}

// Expr is implemented by every node kind. exprNode is unexported so only
// this package can add new node types.
type Expr interface {
	Pos() Pos
	exprNode()
}

type Base struct {
	P Pos
}

func (b Base) Pos() Pos { return b.P }
func (Base) exprNode()  {}

// ---- literals ----

type IntLit struct {
	Base
	Value int64
}

type FloatLit struct {
	Base
	Value float64
}

type BoolLit struct {
	Base
	Value bool
}

type UnitLit struct {
	Base
}

type StringLit struct {
	Base
	Value string
}

type Ident struct {
	Base
	Name string

	// Filled in by the resolver pass, not the parser. Depth is how many scopes
	// to climb from the use site to the scope that owns the variable, and
	// FrameIndex is its slot within that scope. Depth == -1 means the name
	// didn't resolve to any lexical scope (a builtin, a forward reference, or
	// an import) and must fall back to a by-name lookup at runtime.
	Depth      int
	FrameIndex int
}

// ---- operators ----

type BinaryOp int

const (
	PlusOp BinaryOp = iota
	TimesOp
	DivideOp
	SubtractOp
	ModOp
	PowOp
	ShlOp
	ShrOp
	XorOp
	BitAndOp
	BitOrOp

	AndOp
	OrOp

	EqualOp
	NotEqualOp
	GtOp
	LtOp
	GteOp
	LteOp
)

var binaryOperatorsMapping = map[BinaryOp]string{
	PlusOp:     "+",
	TimesOp:    "*",
	SubtractOp: "-",
	DivideOp:   "/",
	ModOp:      "%",
	PowOp:      "^",
	ShlOp:      "<<",
	ShrOp:      ">>",
	XorOp:      "~",
	BitAndOp:   "&",
	BitOrOp:    "|",
	AndOp:      "and",
	OrOp:       "or",
	EqualOp:    "==",
	NotEqualOp: "!=",
	GtOp:       ">",
	LtOp:       "<",
	GteOp:      ">=",
	LteOp:      "<=",
}

func (op BinaryOp) String() string {
	if name, ok := binaryOperatorsMapping[op]; ok {
		return name
	}

	panic("BinaryOp.String: unhandled operator")
}

type UnaryOp int

const (
	MinusOp UnaryOp = iota
	FactorialOp
	NotOp
	BitNotOp
)

var unaryOperatorsMapping = map[UnaryOp]string{
	MinusOp:     "-",
	FactorialOp: "!",
	NotOp:       "!",
	BitNotOp:    "~",
}

func (op UnaryOp) String() string {
	if name, ok := unaryOperatorsMapping[op]; ok {
		return name
	}

	panic("UnaryOp.String: unhandled operator")
}

type AssignOp int

const (
	Assign AssignOp = iota
	AddAssign
	SubAssign
)

var assignmentOperatorsMapping = map[AssignOp]string{
	Assign:    "=",
	AddAssign: "+=",
	SubAssign: "-=",
}

func (op AssignOp) String() string {
	if name, ok := assignmentOperatorsMapping[op]; ok {
		return name
	}

	panic("AssignOp.String: unhandled operator")
}

// ---- composite nodes ----

type BinaryExpr struct {
	Base
	Op          BinaryOp
	Left, Right Expr
}

type UnaryExpr struct {
	Base
	Op      UnaryOp
	Operand Expr
}

// AssignExpr is `left = right` and its compound forms. Const marks a
// `const name = ...` declaration: it always introduces a *new* binding in the
// current scope (never reassigns an outer one) and that binding then rejects
// every later write. Const is only ever set when Op is Assign and Left is an
// *Ident -- the parser rejects `const a[0] = ...` and `const x += ...`.
type AssignExpr struct {
	Base
	Op          AssignOp
	Const       bool
	Left, Right Expr
}

// FnExpr is an anonymous function literal: \(args) -> body. Functions have
// no name because they are primary values. naming
// happens by assigning them to a variable.
type FnExpr struct {
	Base
	Params []*Ident
	Body   []Expr
}

// CallExpr applies Args to whatever Callee evaluates to. Callee can be any
// expression -- a bare Ident, a DotExpr field (`m.f(...)`), an indexed value
// (`arr[0](...)`), or even a parenthesized function literal
// (`(\() -> 1)()`) -- '(' is a general postfix operator, not something
// special-cased to identifiers.
type CallExpr struct {
	Base
	Callee Expr
	Args   []Expr
}

type WhileExpr struct {
	Base
	Condition Expr // nil means "while { ... }" with no condition
	Body      []Expr
}

// ForOfExpr: `for k, v : target { ... }` iterating an array/map/string.
type ForOfExpr struct {
	Base
	Key    *Ident
	Value  *Ident
	Target Expr
	Body   []Expr
}

// ForRangeExpr covers all three range-for shapes:
//
//	for i : N             -> From holds N (the count), To=nil, By=nil; interpreter reads this as 0..N step 1
//	for i : [from, to]    -> From, To set, By=nil (step 1)
//	for i : [from, to, by] -> From, To, By all set
//
// Which of To/By are nil is exactly how the interpreter distinguishes the three shapes.
type ForRangeExpr struct {
	Base
	Index *Ident
	From  Expr
	To    Expr // nil for the bare `for i : N` form (see interpreter: means 0..N)
	By    Expr // nil unless the 3-element bracket form was used
	Body  []Expr
}

type BreakExpr struct {
	Base
	Value Expr // nil if bare `break`/`break;`
}

type ReturnExpr struct {
	Base
	Value Expr // nil if bare `return`/`return;`
}

type ElifBlock struct {
	Condition Expr
	Body      []Expr
}

type IfExpr struct {
	Base
	Condition Expr
	Then      []Expr
	Elifs     []ElifBlock
	Else      []Expr // nil if no else block
}

type ArrayExpr struct {
	Base
	Items []Expr
}

// MapExpr uses parallel Keys/Values slices instead of a single list
// with key,value pairs like was before in the C version.
type MapExpr struct {
	Base
	Keys   []Expr
	Values []Expr
}

// m = {'hello': \() ->;}; `m.hello`
// When using dot expression, the ident name is gonna be used as the key
type DotExpr struct {
	Base
	Left  Expr
	Index Expr
}

// m = {'hello': \() ->;}; m['hello']
// When using square expression, the expression inside the square
// is gonna be evaluated and used as the key
type SquareExpr struct {
	Base
	Left  Expr
	Index Expr
}

func NewBase(p Pos) Base { return Base{P: p} }
