package interp

import (
	"math"
	"strconv"
	"strings"
	"unsafe"

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

// Value is MCA's dynamic value: a tagged union rather than an interface, so a
// scalar rides inside the struct instead of being boxed onto the heap. That's
// what keeps arithmetic in tight loops from allocating.
//
// num holds the bits of the scalar kinds (and the length for a string); ptr
// holds the string data or the *Array/*Map/*FnValue for the heap kinds, nil
// otherwise. Both are read back through the accessors below -- callers switch
// on Kind first and never touch the fields directly.
type Value struct {
	kind Kind
	num  uint64
	ptr  unsafe.Pointer
}

func (v Value) Kind() Kind { return v.kind }

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

func UnitV() Value { return Value{kind: KUnit} }

func IntV(i int64) Value { return Value{kind: KInt, num: uint64(i)} }

func FloatV(f float64) Value { return Value{kind: KFloat, num: math.Float64bits(f)} }

func BoolV(b bool) Value {
	n := uint64(0)
	if b {
		n = 1
	}
	return Value{kind: KBool, num: n}
}

// StringV borrows s's backing bytes rather than copying them; MCA strings are
// immutable, so that's safe and keeps StringV allocation-free. The empty
// string is special-cased because unsafe.StringData("") yields a pointer that
// must not be dereferenced.
func StringV(s string) Value {
	if len(s) == 0 {
		return Value{kind: KString}
	}
	return Value{kind: KString, num: uint64(len(s)), ptr: unsafe.Pointer(unsafe.StringData(s))}
}

func ArrayV(a *Array) Value { return Value{kind: KArray, ptr: unsafe.Pointer(a)} }

func MapV(m *Map) Value { return Value{kind: KMap, ptr: unsafe.Pointer(m)} }

func FnValV(f *FnValue) Value { return Value{kind: KFn, ptr: unsafe.Pointer(f)} }

// intOf/floatOf/... unwrap a Value already known (typically via expectKind) to
// hold that kind. They don't re-check it, same as the type assertions they
// replaced -- callers validate the kind first.
func intOf(v Value) int64     { return int64(v.num) }
func floatOf(v Value) float64 { return math.Float64frombits(v.num) }
func boolOf(v Value) bool     { return v.num != 0 }

func stringOf(v Value) string {
	if v.num == 0 {
		return ""
	}
	return unsafe.String((*byte)(v.ptr), int(v.num))
}

func arrayOf(v Value) *Array { return (*Array)(v.ptr) }
func mapOf(v Value) *Map     { return (*Map)(v.ptr) }
func fnOf(v Value) *FnValue  { return (*FnValue)(v.ptr) }

// AsInt/AsFloat/... are the exported spelling of the accessors above, for the
// native packages (internal/packages/...) which can't reach the unexported
// ones. Each assumes the Value already holds that kind.
func AsInt(v Value) int64     { return intOf(v) }
func AsFloat(v Value) float64 { return floatOf(v) }
func AsBool(v Value) bool     { return boolOf(v) }
func AsString(v Value) string { return stringOf(v) }
func AsArray(v Value) *Array  { return arrayOf(v) }
func AsMap(v Value) *Map      { return mapOf(v) }

func Truthy(v Value) (result bool, ok bool) {
	switch v.kind {
	case KMap:
		return mapOf(v).Len() > 0, true
	case KArray:
		return len(arrayOf(v).Items) > 0, true
	case KUnit:
		return false, true
	case KInt:
		return v.num != 0, true
	case KFloat:
		return floatOf(v) != 0, true
	case KBool:
		return v.num != 0, true
	case KString:
		return v.num > 0, true
	case KFn:
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

// FormatFloat renders a float the way MCA prints and stringifies it: the
// shortest decimal string that round-trips to the same float64, so an exact
// value like 1.32 shows as "1.32" rather than "1.320000". It never uses
// scientific notation. A whole-valued float keeps a trailing ".0" so it stays
// visually distinct from an int, while NaN and ±Inf render as themselves.
func FormatFloat(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}

	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return s
}
