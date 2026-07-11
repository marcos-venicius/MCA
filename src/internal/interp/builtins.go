package interp

import (
	"fmt"
	"math"

	"mca/internal/ast"
)

// arity wraps a BuiltinFn with a fixed-arity check, run before the
// builtin's own implementation. n == -1 means variadic (no check).
func arity(name string, n int, fn BuiltinFn) BuiltinFn {
	if n < 0 {
		return fn
	}

	return func(in *Interp, caller ast.Expr, args []ast.Expr) Value {
		if len(args) > n {
			throw(caller.Pos(), "too many arguments %s(...). expected %d but got %d", name, n, len(args))
		} else if len(args) < n {
			throw(caller.Pos(), "too few arguments %s(...). expected %d but got %d", name, n, len(args))
		}
		return fn(in, caller, args)
	}
}

// builtins is the flat registration table shared by every Interp instance.
// Declared without an initializer and populated from init() rather than a
// `var builtins = map[...]{...}` literal: nearly every builtin's body calls
// in.Eval, which dispatches back through evalCall's builtins lookup -- a
// real call-graph cycle, but not a *value* cycle, since none of it runs
// during package initialization. Go's initializer-cycle check can't tell the
// difference for a direct var initializer and rejects it regardless; moving
// the literal into init() (a plain function body, not a variable initializer
// expression) sidesteps that static check entirely.
var builtins map[string]BuiltinFn

func init() {
	builtins = map[string]BuiltinFn{
		// Math related
		"PI":    arity("PI", 0, builtinPI),
		"E":     arity("E", 0, builtinE),
		"abs":   arity("abs", 1, builtinAbs),
		"max":   arity("max", -1, builtinMax),
		"min":   arity("min", -1, builtinMin),
		"sin":   arity("sin", 1, mathBuiltin(math.Sin)),
		"cos":   arity("cos", 1, mathBuiltin(math.Cos)),
		"asin":  arity("asin", 1, mathBuiltin(math.Asin)),
		"acos":  arity("acos", 1, mathBuiltin(math.Acos)),
		"tan":   arity("tan", 1, mathBuiltin(math.Tan)),
		"rad":   arity("rad", 1, mathBuiltin(calcRad)),
		"deg":   arity("deg", 1, mathBuiltin(calcDeg)),
		"sqrt":  arity("sqrt", 1, mathBuiltin(math.Sqrt)),
		"log":   arity("log", 1, mathBuiltin(math.Log)),
		"log10": arity("log10", 1, mathBuiltin(math.Log10)),
		"exp":   arity("exp", 1, mathBuiltin(math.Exp)),
		"floor": arity("floor", 1, mathBuiltin(math.Floor)),
		"ceil":  arity("ceil", 1, mathBuiltin(math.Ceil)),
		"round": arity("round", 1, mathBuiltin(math.Round)),

		// I/O / System related
		"println":          arity("println", -1, builtinPrintln),
		"print":            arity("print", -1, builtinPrint),
		"read_entire_file": arity("read_entire_file", 1, builtinReadEntireFile),
		"exit":             arity("exit", 1, builtinExit),

		// language specifics
		"import": arity("import", 1, builtinImport),

		"type":      arity("type", 1, builtinType),
		"argc":      arity("argc", 0, builtinArgc),
		"argv":      arity("argv", 1, builtinArgv),
		"as_int":    arity("as_int", 1, builtinAsInt),
		"as_float":  arity("as_float", 1, builtinAsFloat),
		"as_bool":   arity("as_bool", 1, builtinAsBool),
		"as_string": arity("as_string", 1, builtinAsString),
		"is_int":    arity("is_int", 1, isTypeBuiltin(KInt)),
		"is_float":  arity("is_float", 1, isTypeBuiltin(KFloat)),
		"is_bool":   arity("is_bool", 1, isTypeBuiltin(KBool)),
		"is_string": arity("is_string", 1, isTypeBuiltin(KString)),
		"is_unit":   arity("is_unit", 1, isTypeBuiltin(KUnit)),
		"is_array":  arity("is_array", 1, isTypeBuiltin(KArray)),
		"is_map":    arity("is_map", 1, isTypeBuiltin(KMap)),
		"is_fn":     arity("is_fn", 1, isTypeBuiltin(KFn)),
		"len":       arity("len", 1, builtinLen),

		// Strings
		"repeat":      arity("repeat", 2, builtinRepeat),
		"replace":     arity("replace", 3, builtinReplace),
		"starts_with": arity("starts_with", 2, builtinStartsWith),
		"ends_with":   arity("ends_with", 2, builtinEndsWith),
		"lower":       arity("lower", 1, builtinLower),
		"upper":       arity("upper", 1, builtinUpper),
		"trim":        arity("trim", 1, builtinTrim),
		"ltrim":       arity("ltrim", 1, builtinLTrim),
		"rtrim":       arity("rtrim", 1, builtinRTrim),
		"join":        arity("join", 2, builtinJoin),
		"split":       arity("split", 2, builtinSplit),
		"select":      arity("select", 3, builtinSelect),
		"ord":         arity("ord", 1, builtinOrd),
		"chr":         arity("chr", 1, builtinChr),
		"format":      arity("format", -1, builtinFormat),

		// Maps
		"keys":      arity("keys", 1, builtinMapKeys),
		"values":    arity("values", 1, builtinMapValues),
		"map_del":   arity("map_del", 2, builtinMapDel),
		"map_clear": arity("map_clear", 1, builtinMapClear),

		// Arrays
		"contains": arity("contains", 2, builtinContains),
		"map":      arity("map", 2, builtinMap),
		"filter":   arity("filter", 2, builtinFilter),
		"append":   arity("append", 2, builtinAppend),

		// random
		"srand": arity("srand", 1, builtinSrand),
		"rand":  arity("rand", 2, builtinRand),

		// TODO: may I have a 'Date' value type?
		// datetime related
		"time":        arity("time", 0, builtinTime),
		"year":        arity("year", 1, builtinYear),
		"month":       arity("month", 1, builtinMonth),
		"date":        arity("date", 1, builtinDate),
		"day":         arity("day", 1, builtinDay),
		"hour":        arity("hour", 1, builtinHour),
		"minute":      arity("minute", 1, builtinMinute),
		"second":      arity("second", 1, builtinSecond),
		"millisecond": arity("millisecond", 0, builtinMillisecond),
	}
}

// ---- printing helpers ----

func printValue(in *Interp, v Value, wrapStrings bool) {
	switch vv := v.(type) {
	case IntValue:
		fmt.Fprintf(in.Out, "%d", int64(vv))
	case FloatValue:
		fmt.Fprintf(in.Out, "%f", float64(vv))
	case BoolValue:
		if vv {
			fmt.Fprint(in.Out, "true")
		} else {
			fmt.Fprint(in.Out, "false")
		}
	case UnitValue:
		fmt.Fprint(in.Out, "(unit)")
	case StringValue:
		if wrapStrings {
			fmt.Fprintf(in.Out, "'%s'", string(vv))
		} else {
			fmt.Fprint(in.Out, string(vv))
		}
	case *Map:
		printMap(in, vv)
	case *FnValue:
		fmt.Fprintf(in.Out, "fn(...%d)", len(vv.Node.Params))
	case *Array:
		printArray(in, vv)
	}
}

func printArray(in *Interp, a *Array) {
	fmt.Fprint(in.Out, "[")
	for i, item := range a.Items {
		if i > 0 {
			fmt.Fprint(in.Out, ", ")
		}
		printValue(in, item, true)
	}
	fmt.Fprint(in.Out, "]")
}

func printMap(in *Interp, m *Map) {
	fmt.Fprint(in.Out, "{")
	i := 0
	for k, v := range m.values {
		if i > 0 {
			fmt.Fprint(in.Out, ", ")
		}
		if k.Kind == KInt {
			printValue(in, IntV(k.I), true)
		} else {
			printValue(in, StringV(k.S), true)
		}
		fmt.Fprint(in.Out, ": ")
		printValue(in, v, true)

		i++
	}
	fmt.Fprint(in.Out, "}")
}
