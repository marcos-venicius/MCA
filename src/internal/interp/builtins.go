package interp

import (
	"fmt"
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
		// Math related (everything else numeric lives in the 'math' package)
		"sum": native("sum", 1, builtinSum),
		"max": native("max", -1, builtinMax),
		"min": native("min", -1, builtinMin),

		// I/O / System related (file access lives in the 'io' package)
		"println": native("println", -1, builtinPrintln),
		"print":   native("print", -1, builtinPrint),
		"exit":    native("exit", 1, builtinExit),

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

		// Maps (text manipulation lives in the 'string' package)
		"keys":   native("keys", 1, builtinMapKeys),
		"values": native("values", 1, builtinMapValues),

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
	switch v.Kind() {
	case KInt:
		fmt.Fprintf(in.Out, "%d", intOf(v))
	case KFloat:
		fmt.Fprintf(in.Out, "%f", floatOf(v))
	case KBool:
		if boolOf(v) {
			fmt.Fprint(in.Out, "true")
		} else {
			fmt.Fprint(in.Out, "false")
		}
	case KUnit:
		fmt.Fprint(in.Out, "(unit)")
	case KString:
		if wrapStrings {
			fmt.Fprintf(in.Out, "'%s'", stringOf(v))
		} else {
			fmt.Fprint(in.Out, stringOf(v))
		}
	case KMap:
		printMap(in, mapOf(v))
	case KFn:
		fv := fnOf(v)
		if fv.Native != nil {
			fmt.Fprintf(in.Out, "builtin %s(...)", fv.Native.Name)
		} else {
			fmt.Fprintf(in.Out, "fn(...%d)", len(fv.Node.Params))
		}
	case KArray:
		printArray(in, arrayOf(v))
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
		printValue(in, mapValueFromKey(k), true)
		fmt.Fprint(in.Out, ": ")
		printValue(in, v, true)

		i++
	}
	fmt.Fprint(in.Out, "}")
}
