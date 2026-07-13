package interp

import (
	"fmt"
	"math"
)

// native describes one builtin. n is its exact argument count, enforced by
// the shared call path before the implementation runs; n == -1 means variadic
// (the builtin checks its own argument count).
func native(name string, n int, fn BuiltinFn) *Native {
	return &Native{Name: name, Arity: n, Fn: fn}
}

// builtins is the flat registration table every Interp is seeded from: New()
// binds each entry into the global scope as a constant, so builtins are
// ordinary values rather than a side table consulted at call time.
//
// Declared without an initializer and populated from init() rather than a
// `var builtins = map[...]{...}` literal: nearly every builtin's body calls
// back into in.Eval / in.callFnValue -- a real call-graph cycle, but not a
// *value* cycle, since none of it runs during package initialization. Go's
// initializer-cycle check can't tell the difference for a direct var
// initializer and rejects it regardless; moving the literal into init() (a
// plain function body, not a variable initializer expression) sidesteps that
// static check entirely.
var builtins map[string]*Native

func init() {
	builtins = map[string]*Native{
		// Math related
		"PI":    native("PI", 0, builtinPI),
		"E":     native("E", 0, builtinE),
		"sum":   native("sum", 1, builtinSum),
		"abs":   native("abs", 1, builtinAbs),
		"max":   native("max", -1, builtinMax),
		"min":   native("min", -1, builtinMin),
		"sin":   native("sin", 1, mathBuiltin(math.Sin)),
		"cos":   native("cos", 1, mathBuiltin(math.Cos)),
		"asin":  native("asin", 1, mathBuiltin(math.Asin)),
		"acos":  native("acos", 1, mathBuiltin(math.Acos)),
		"tan":   native("tan", 1, mathBuiltin(math.Tan)),
		"rad":   native("rad", 1, mathBuiltin(calcRad)),
		"deg":   native("deg", 1, mathBuiltin(calcDeg)),
		"sqrt":  native("sqrt", 1, mathBuiltin(math.Sqrt)),
		"log":   native("log", 1, mathBuiltin(math.Log)),
		"log10": native("log10", 1, mathBuiltin(math.Log10)),
		"exp":   native("exp", 1, mathBuiltin(math.Exp)),
		"floor": native("floor", 1, mathBuiltin(math.Floor)),
		"ceil":  native("ceil", 1, mathBuiltin(math.Ceil)),
		"round": native("round", 1, mathBuiltin(math.Round)),

		// I/O / System related
		"println":          native("println", -1, builtinPrintln),
		"print":            native("print", -1, builtinPrint),
		"read_entire_file": native("read_entire_file", 1, builtinReadEntireFile),
		"exit":             native("exit", 1, builtinExit),

		// language specifics
		"import": native("import", 1, builtinImport),
		"help":   native("help", -1, builtinHelp),

		"type":      native("type", 1, builtinType),
		"argc":      native("argc", 0, builtinArgc),
		"argv":      native("argv", 1, builtinArgv),
		"as_int":    native("as_int", 1, builtinAsInt),
		"as_float":  native("as_float", 1, builtinAsFloat),
		"as_bool":   native("as_bool", 1, builtinAsBool),
		"as_string": native("as_string", 1, builtinAsString),
		"is_int":    native("is_int", 1, isTypeBuiltin(KInt)),
		"is_float":  native("is_float", 1, isTypeBuiltin(KFloat)),
		"is_bool":   native("is_bool", 1, isTypeBuiltin(KBool)),
		"is_string": native("is_string", 1, isTypeBuiltin(KString)),
		"is_unit":   native("is_unit", 1, isTypeBuiltin(KUnit)),
		"is_array":  native("is_array", 1, isTypeBuiltin(KArray)),
		"is_map":    native("is_map", 1, isTypeBuiltin(KMap)),
		"is_fn":     native("is_fn", 1, isTypeBuiltin(KFn)),
		"len":       native("len", 1, builtinLen),

		// Strings
		"repeat":      native("repeat", 2, builtinRepeat),
		"replace":     native("replace", 3, builtinReplace),
		"starts_with": native("starts_with", 2, builtinStartsWith),
		"ends_with":   native("ends_with", 2, builtinEndsWith),
		"lower":       native("lower", 1, builtinLower),
		"upper":       native("upper", 1, builtinUpper),
		"trim":        native("trim", 1, builtinTrim),
		"ltrim":       native("ltrim", 1, builtinLTrim),
		"rtrim":       native("rtrim", 1, builtinRTrim),
		"join":        native("join", 2, builtinJoin),
		"split":       native("split", 2, builtinSplit),
		"select":      native("select", 3, builtinSelect),
		"ord":         native("ord", 1, builtinOrd),
		"chr":         native("chr", 1, builtinChr),
		"format":      native("format", -1, builtinFormat),

		// Maps
		"keys":      native("keys", 1, builtinMapKeys),
		"values":    native("values", 1, builtinMapValues),
		"map_del":   native("map_del", 2, builtinMapDel),
		"map_clear": native("map_clear", 1, builtinMapClear),

		// Arrays
		"indexes_to_keys": native("indexes_to_keys", 2, builtinIndexesToKeys),
		"sort":            native("sort", 2, builtinSort),
		"reverse":         native("reverse", 1, builtinReverse),
		"concat":          native("concat", -1, builtinConcat),
		"contains":        native("contains", 2, builtinContains),
		"map":             native("map", 2, builtinMap),
		"filter":          native("filter", 2, builtinFilter),
		"append":          native("append", 2, builtinAppend),
		"delete":          native("delete", -1, builtinDelete),

		// random
		"srand": native("srand", 1, builtinSrand),
		"rand":  native("rand", 2, builtinRand),

		// TODO: may I have a 'Date' value type?
		// datetime related
		"time":        native("time", 0, builtinTime),
		"year":        native("year", 1, builtinYear),
		"month":       native("month", 1, builtinMonth),
		"date":        native("date", 1, builtinDate),
		"day":         native("day", 1, builtinDay),
		"hour":        native("hour", 1, builtinHour),
		"minute":      native("minute", 1, builtinMinute),
		"second":      native("second", 1, builtinSecond),
		"millisecond": native("millisecond", 0, builtinMillisecond),
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
		if vv.Native != nil {
			fmt.Fprintf(in.Out, "builtin %s(...)", vv.Native.Name)
		} else {
			fmt.Fprintf(in.Out, "fn(...%d)", len(vv.Node.Params))
		}
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
