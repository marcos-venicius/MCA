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
)

var unaryOperatorsMapping = map[UnaryOp]string{
	MinusOp:     "-",
	FactorialOp: "!",
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

type AssignExpr struct {
	Base
	Op          AssignOp
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

// CallExpr only ever carries a bare function name, never a receiver -- calls
// are recognized in exactly one parser position (an Ident immediately
// followed by '('), so there is no general "call any expression" rule.
// e.g. arr[0](5) does NOT parse as a call. TODO: Fix this behavior?
type CallExpr struct {
	Base
	FnName string
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

// IndexExpr backs both `left[index]` and `left.index` -- the parser
// compiles both forms into this same node, and
// the interpreter tells them apart at runtime by what Index actually is
// (an arbitrary Expr for `[]`, always an *Ident or *CallExpr for `.`).
type IndexExpr struct {
	Base
	Left  Expr
	Index Expr
}

func NewBase(p Pos) Base { return Base{P: p} }
