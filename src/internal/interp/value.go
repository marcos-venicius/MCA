package interp

import (
	"strings"

	"mca/internal/ast"
)

type Kind int

const (
	KUnit Kind = iota
	KInt
	KFloat
	KBool
	KString
	KArray
	KMap
	KFn
)

var valueKindMappings = map[Kind]string{
	KUnit:   "unit",
	KInt:    "int",
	KFloat:  "float",
	KBool:   "bool",
	KString: "string",
	KArray:  "array",
	KMap:    "map",
	KFn:     "fn",
}

func (k Kind) String() string {
	if name, ok := valueKindMappings[k]; ok {
		return name
	}

	panic("Kind.String: unhandled kind")
}

// Value is MCA's single dynamic value representation: any Go type that knows
// its own Kind. Each concrete kind gets its own small type (IntValue,
// FloatValue, ...) rather than one struct carrying every possible field at
// once, so a value only ever holds the data it actually needs.
type Value interface {
	Kind() Kind
}

type UnitValue struct{}

func (UnitValue) Kind() Kind { return KUnit }

type IntValue int64

func (IntValue) Kind() Kind { return KInt }

type FloatValue float64

func (FloatValue) Kind() Kind { return KFloat }

type BoolValue bool

func (BoolValue) Kind() Kind { return KBool }

type StringValue string

func (StringValue) Kind() Kind { return KString }

// Native is a builtin's implementation, wrapped so it can be handed around as
// an ordinary MCA value. Arity is the exact argument count the shared call
// path enforces before dispatching; -1 means the builtin is variadic and
// checks its own argument count.
type Native struct {
	Name  string
	Arity int
	Fn    BuiltinFn
}

// FnValue is every callable in MCA: either a user function literal or a
// builtin.
//
// For a user function it pairs the literal with the environment captured at
// the moment *that evaluation* of the literal ran. Each evaluation of a
// \(...)-> literal produces its own FnValue with its own captured Env, so two
// values produced from the same literal (e.g. across loop iterations) never
// alias each other's closure state.
//
// For a builtin, Native is set and Node/Env are nil. Both shapes share one
// Kind (KFn) and one Go type deliberately: it is what lets a builtin be
// stored, passed to map/filter/sort, and called through exactly the same
// paths as anything the user wrote.
type FnValue struct {
	Node   *ast.FnExpr // nil for builtins
	Env    *Env        // nil for builtins
	Native *Native     // nil for user functions
}

func (*FnValue) Kind() Kind { return KFn }

// Arity is how many arguments the function takes; -1 for a variadic builtin.
func (f *FnValue) Arity() int {
	if f.Native != nil {
		return f.Native.Arity
	}
	return len(f.Node.Params)
}

// Accepts reports whether the function can be called with n arguments. A
// variadic builtin accepts any count, which is what lets `map(arr, println)`
// work alongside `map(arr, upper)`.
func (f *FnValue) Accepts(n int) bool {
	a := f.Arity()
	return a < 0 || a == n
}

func UnitV() Value             { return UnitValue{} }
func IntV(i int64) Value       { return IntValue(i) }
func FloatV(f float64) Value   { return FloatValue(f) }
func BoolV(b bool) Value       { return BoolValue(b) }
func StringV(s string) Value   { return StringValue(s) }
func FnValV(fv *FnValue) Value { return fv }
func ArrayV(a *Array) Value    { return a }
func MapV(m *Map) Value        { return m }

// intOf/floatOf/boolOf/stringOf unwrap a Value already known (typically via
// expectKind) to hold that concrete kind. They panic on mismatch, same as any
// other unchecked type assertion -- callers are expected to have validated
// the kind first.
func intOf(v Value) int64     { return int64(v.(IntValue)) }
func floatOf(v Value) float64 { return float64(v.(FloatValue)) }
func boolOf(v Value) bool     { return bool(v.(BoolValue)) }
func stringOf(v Value) string { return string(v.(StringValue)) }

func Truthy(v Value) (result bool, ok bool) {
	switch vv := v.(type) {
	case *Map:
		return vv.Len() > 0, true
	case *Array:
		return len(vv.Items) > 0, true
	case UnitValue:
		return false, true
	case IntValue:
		return vv != 0, true
	case FloatValue:
		return vv != 0, true
	case BoolValue:
		return bool(vv), true
	case StringValue:
		return len(vv) > 0, true
	case *FnValue:
		return true, true
	default:
		return false, false
	}
}

// KindsName joins kind names with " | ", for "expected a 'X | Y' but got a
// 'Z'" diagnostics.
func KindsName(kinds ...Kind) string {
	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = k.String()
	}
	return strings.Join(names, " | ")
}
