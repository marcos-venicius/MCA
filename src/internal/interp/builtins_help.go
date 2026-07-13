package interp

import (
	"sort"
	"strings"
)

// Param is one documented parameter of a builtin. Variadic marks the
// last parameter of a variadic builtin (e.g. max, println, format) -- it
// renders with a trailing "..." and may be supplied zero or more times
// (the Description says whether a minimum count applies).
type Param struct {
	Name     string
	Type     string
	Variadic bool
}

type Doc struct {
	Params      []Param
	Returns     string
	Description string
	Examples    []string
}

func p(name, typ string) Param  { return Param{Name: name, Type: typ} }
func pv(name, typ string) Param { return Param{Name: name, Type: typ, Variadic: true} }

// helpCategories groups builtin names for help()'s no-argument overview,
// mirroring the section comments in builtins.go's registration table.
var helpCategories = []struct {
	Name  string
	Funcs []string
}{
	{"Math", []string{
		"PI", "E", "sum", "abs", "max", "min", "sin", "cos", "asin", "acos", "tan",
		"rad", "deg", "sqrt", "log", "log10", "exp", "floor", "ceil", "round",
	}},
	{"I/O & System", []string{
		"println", "print", "read_entire_file", "exit", "argc", "argv",
	}},
	{"Language", []string{
		"import", "type", "is_int", "is_float", "is_bool", "is_string",
		"is_unit", "is_array", "is_map", "is_fn", "len", "help",
	}},
	{"Casting", []string{
		"as_int", "as_float", "as_bool", "as_string",
	}},
	{"Strings", []string{
		"repeat", "replace", "starts_with", "ends_with", "lower", "upper",
		"trim", "ltrim", "rtrim", "join", "split", "select", "ord", "chr", "format",
	}},
	{"Maps", []string{
		"keys", "values", "map_del", "map_clear",
	}},
	{"Arrays", []string{
		"concat", "contains", "map", "filter", "append", "delete", "reverse", "sort",
		"indexes_to_keys",
	}},
	{"Random", []string{
		"srand", "rand",
	}},
	{"Date & Time", []string{
		"time", "year", "month", "date", "day", "hour", "minute", "second", "millisecond",
	}},
}

var helpDocs = map[string]Doc{
	// ---- Math ----

	"PI": {
		Returns:     "float",
		Description: "The constant pi (3.14159...).",
		Examples:    []string{`PI()  ->  3.141593`},
	},
	"E": {
		Returns:     "float",
		Description: "The constant e (2.71828...).",
		Examples:    []string{`E()  ->  2.718282`},
	},
	"abs": {
		Params:      []Param{p("x", "int|float|bool")},
		Returns:     "int|float",
		Description: "Absolute value of x. Returns an int if x is an int or bool, a float if x is a float.",
		Examples: []string{
			`abs(-15.5)  ->  15.5`,
			`abs(-1)  ->  1`,
		},
	},
	"sum": {
		Params:      []Param{p("arr", "array")},
		Returns:     "int|float",
		Description: "Sum of every element of arr. Every element must be an int or a float (unlike abs/max/min, bools aren't accepted). Stays int if every element is an int, otherwise promotes to float. Sums to 0 for an empty array.",
		Examples: []string{
			`sum([1, 2, 3])  ->  6`,
			`sum([1, 2.5])  ->  3.5`,
		},
	},
	"max": {
		Params:      []Param{pv("values", "int|float|bool")},
		Returns:     "int|float",
		Description: "Largest of one or more numbers. Requires at least one argument. Stays int if every argument is an int, otherwise promotes to float.",
		Examples: []string{
			`max(10.5, 20.0)  ->  20.0`,
			`max(3, 8, 1)  ->  8`,
		},
	},
	"min": {
		Params:      []Param{pv("values", "int|float|bool")},
		Returns:     "int|float",
		Description: "Smallest of one or more numbers. Requires at least one argument. Stays int if every argument is an int, otherwise promotes to float.",
		Examples: []string{
			`min(10.5, 20.0)  ->  10.5`,
			`min(3, 8, 1)  ->  1`,
		},
	},
	"sin":  mathDoc("sin", "Sine of x, in radians."),
	"cos":  mathDoc("cos", "Cosine of x, in radians."),
	"asin": mathDoc("asin", "Arcsine of x, result in radians."),
	"acos": mathDoc("acos", "Arccosine of x, result in radians."),
	"tan":  mathDoc("tan", "Tangent of x, in radians."),
	"rad": {
		Params:      []Param{p("degrees", "int|float|bool")},
		Returns:     "int|float",
		Description: "Converts degrees to radians.",
		Examples:    []string{`rad(180)  ->  3.141593`},
	},
	"deg": {
		Params:      []Param{p("radians", "int|float|bool")},
		Returns:     "int|float",
		Description: "Converts radians to degrees.",
		Examples:    []string{`deg(PI())  ->  180`},
	},
	"sqrt":  mathDoc("sqrt", "Square root of x."),
	"log":   mathDoc("log", "Natural logarithm (base e) of x."),
	"log10": mathDoc("log10", "Base-10 logarithm of x."),
	"exp":   mathDoc("exp", "e raised to the power of x."),
	"floor": mathDoc("floor", "x rounded down to the nearest whole number."),
	"ceil":  mathDoc("ceil", "x rounded up to the nearest whole number."),
	"round": mathDoc("round", "x rounded to the nearest whole number (half away from zero)."),

	// ---- I/O & System ----

	"println": {
		Params:      []Param{pv("values", "any")},
		Returns:     "any",
		Description: "Prints zero or more values separated by a space, followed by a newline, to standard output. Returns the last value printed (or unit if called with no arguments).",
		Examples: []string{
			`println('hello', 'world')  -- prints 'hello world' then a newline`,
			`println()  -- prints just a newline`,
		},
	},
	"print": {
		Params:      []Param{pv("values", "any")},
		Returns:     "any",
		Description: "Prints zero or more values with no separator and no trailing newline, to standard output. Returns the last value printed (or unit if called with no arguments).",
		Examples:    []string{`print('a', 'b', 'c')  -- prints: abc`},
	},
	"read_entire_file": {
		Params:      []Param{p("path", "string")},
		Returns:     "string",
		Description: "Reads the whole contents of the file at path and returns it as a string. Throws a runtime error if the file cannot be read.",
		Examples:    []string{`read_entire_file('data.txt')`},
	},
	"exit": {
		Params:      []Param{p("code", "int")},
		Returns:     "(never returns)",
		Description: "Immediately terminates the process with the given exit code.",
		Examples:    []string{`exit(1)`},
	},
	"argc": {
		Returns:     "int",
		Description: "Number of command-line arguments passed to the script (Args[0] is conventionally the script's own path).",
		Examples:    []string{`argc()`},
	},
	"argv": {
		Params:      []Param{p("index", "int")},
		Returns:     "string",
		Description: "The command-line argument at index. Throws a runtime error if index is out of range.",
		Examples:    []string{`argv(0)  -- the script path`},
	},

	// ---- Language ----

	"import": {
		Params:      []Param{p("path", "string")},
		Returns:     "any",
		Description: "Loads, parses, and runs the .mca file at path as a fresh module (its own isolated environment) and returns the value of its last top-level expression -- typically a map used as the module's exported interface. A path starting with '.' resolves relative to the importing file's own directory.",
		Examples:    []string{`u = import('./utils.mca'); u.double(21)`},
	},
	"type": {
		Params:      []Param{p("value", "any")},
		Returns:     "string",
		Description: `Name of value's kind, one of: "unit", "int", "float", "bool", "string", "array", "map", "fn".`,
		Examples: []string{
			`type(4)  ->  'int'`,
			`type(4.4)  ->  'float'`,
		},
	},
	"is_int":    isDoc("int"),
	"is_float":  isDoc("float"),
	"is_bool":   isDoc("bool"),
	"is_string": isDoc("string"),
	"is_unit":   isDoc("unit"),
	"is_array":  isDoc("array"),
	"is_map":    isDoc("map"),
	"is_fn": {
		Params:      []Param{p("value", "any")},
		Returns:     "bool",
		Description: "True if value is a function (a closure created with \\(...) -> ...). Builtins are not first-class values, so this never applies to them.",
		Examples:    []string{`is_fn(\(x) -> x)  ->  true`},
	},
	"len": {
		Params:      []Param{p("value", "string|array|map")},
		Returns:     "int",
		Description: "Length of value: byte length for a string, item count for an array, key count for a map.",
		Examples: []string{
			`len('Hello World')  ->  11`,
			`len([1, 2, 3])  ->  3`,
		},
	},
	"help": {
		Params:      []Param{p("name", "string")},
		Returns:     "unit",
		Description: "Prints documentation for a builtin function: its signature, parameter types, return type, a description, and usage examples. name is optional -- called with no arguments, prints an overview listing every builtin grouped by category instead. Only builtins are documented -- user-defined functions have no help entry.",
		Examples: []string{
			`help(map)`,
			`help()`,
		},
	},

	// ---- Casting ----

	"as_int": {
		Params:      []Param{p("value", "int|float|bool|string")},
		Returns:     "int",
		Description: "Casts value to int: floats truncate toward zero, bools become 0/1, strings are parsed as base-10 integers. Throws a runtime error if a string isn't a valid integer literal or the value is out of range.",
		Examples:    []string{`as_int('42')  ->  42`},
	},
	"as_float": {
		Params:      []Param{p("value", "int|float|bool|string")},
		Returns:     "float",
		Description: "Casts value to float: bools become 0.0/1.0, strings are parsed as floating-point literals. Throws a runtime error if a string isn't a valid float literal or the value is out of range.",
		Examples:    []string{`as_float('3.14')  ->  3.14`},
	},
	"as_bool": {
		Params:      []Param{p("value", "any")},
		Returns:     "bool",
		Description: "Casts value to bool using the same truthiness rules as if/while: unit is false; nonzero int/float is true; a bool is itself; a nonempty string, array, or map is true; a function is always true.",
		Examples: []string{
			`as_bool(0)  ->  false`,
			`as_bool('hi')  ->  true`,
		},
	},
	"as_string": {
		Params:      []Param{p("value", "int|float|bool|string")},
		Returns:     "string",
		Description: "Casts value to string. Floats are formatted with 6 digits after the decimal point. Arrays and maps aren't supported yet.",
		Examples: []string{
			`as_string(42)  ->  '42'`,
			`as_string(true)  ->  'true'`,
		},
	},

	// ---- Strings ----

	"repeat": {
		Params:      []Param{p("str", "string"), p("count", "int")},
		Returns:     "string",
		Description: "str repeated count times, concatenated with no separator. Throws a runtime error if count is negative.",
		Examples:    []string{`repeat('ab', 3)  ->  'ababab'`},
	},
	"replace": {
		Params:      []Param{p("str", "string"), p("old", "string"), p("new", "string")},
		Returns:     "string",
		Description: "str with every non-overlapping occurrence of old replaced by new.",
		Examples: []string{
			`replace('Hello World', 'World', 'There')  ->  'Hello There'`,
			`replace('aaa', 'a', 'b')  ->  'bbb'`,
		},
	},
	"starts_with": {
		Params:      []Param{p("str", "string"), p("prefix", "string")},
		Returns:     "bool",
		Description: "True if str starts with prefix. Case-sensitive; an empty prefix always matches.",
		Examples:    []string{`starts_with('Hello World', 'Hello')  ->  true`},
	},
	"ends_with": {
		Params:      []Param{p("str", "string"), p("suffix", "string")},
		Returns:     "bool",
		Description: "True if str ends with suffix. Case-sensitive; an empty suffix always matches.",
		Examples:    []string{`ends_with('Hello World', 'World')  ->  true`},
	},
	"lower": {
		Params:      []Param{p("str", "string")},
		Returns:     "string",
		Description: "str with every letter lowercased.",
		Examples:    []string{`lower('HELLO')  ->  'hello'`},
	},
	"upper": {
		Params:      []Param{p("str", "string")},
		Returns:     "string",
		Description: "str with every letter uppercased.",
		Examples:    []string{`upper('hello')  ->  'HELLO'`},
	},
	"trim": {
		Params:      []Param{p("str", "string")},
		Returns:     "string",
		Description: "str with leading and trailing whitespace removed.",
		Examples:    []string{`trim('  hello  ')  ->  'hello'`},
	},
	"ltrim": {
		Params:      []Param{p("str", "string")},
		Returns:     "string",
		Description: "str with leading whitespace removed.",
		Examples:    []string{`ltrim('  hello  ')  ->  'hello  '`},
	},
	"rtrim": {
		Params:      []Param{p("str", "string")},
		Returns:     "string",
		Description: "str with trailing whitespace removed.",
		Examples:    []string{`rtrim('  hello  ')  ->  '  hello'`},
	},
	"join": {
		Params:      []Param{p("items", "array"), p("sep", "string")},
		Returns:     "string",
		Description: "Concatenates items (every element must be a string) with sep placed between each pair.",
		Examples:    []string{`join(['a', 'b', 'c'], ',')  ->  'a,b,c'`},
	},
	"split": {
		Params:      []Param{p("str", "string"), p("sep", "string")},
		Returns:     "array",
		Description: "Splits str on every occurrence of sep into an array of strings. If sep doesn't occur, returns a single-element array holding the whole string.",
		Examples:    []string{`split('a,b,c', ',')  ->  ['a', 'b', 'c']`},
	},
	"select": {
		Params:      []Param{p("str", "string"), p("from", "int"), p("to", "int")},
		Returns:     "string",
		Description: "Byte substring of str from index from (inclusive) to index to (exclusive). Throws a runtime error if the range is out of bounds or from > to.",
		Examples:    []string{`select('Hello, World', 7, 12)  ->  'World'`},
	},
	"ord": {
		Params:      []Param{p("char", "string")},
		Returns:     "int",
		Description: "Byte value of char, which must be a string of length exactly 1.",
		Examples:    []string{`ord('a')  ->  97`},
	},
	"chr": {
		Params:      []Param{p("codepoint", "int")},
		Returns:     "string",
		Description: "The UTF-8 string for the Unicode codepoint. Codepoints outside the valid range fall back to the replacement character (U+FFFD) rather than erroring.",
		Examples:    []string{`chr(65)  ->  'A'`},
	},
	"format": {
		Params:      []Param{pv("values", "int|string|float|bool")},
		Returns:     "string",
		Description: "Concatenates one or more values into a single string with no separator (ints/floats/bools are stringified). Requires at least one argument.",
		Examples:    []string{`format('I am ', 5, ' years old')  ->  'I am 5 years old'`},
	},

	// ---- Maps ----

	"keys": {
		Params:      []Param{p("m", "map")},
		Returns:     "array",
		Description: "A new array holding every key in m. Iteration order isn't guaranteed and is randomized independently on every call -- keys(m) and a separate values(m) call are not guaranteed to line up positionally. Use `for k, v : m` for paired iteration.",
		Examples:    []string{`m = {'a': 1, 'b': 2}; keys(m)  ->  ['a', 'b']  (order not guaranteed)`},
	},
	"values": {
		Params:      []Param{p("m", "map")},
		Returns:     "array",
		Description: "A new array holding every value in m. Iteration order isn't guaranteed and is randomized independently on every call -- see keys().",
		Examples:    []string{`m = {'a': 1, 'b': 2}; values(m)  ->  [1, 2]  (order not guaranteed)`},
	},
	"map_del": {
		Params:      []Param{p("m", "map"), p("key", "int|string")},
		Returns:     "bool",
		Description: "Removes key from m. Returns true if the key existed (and was removed), false if it wasn't present.",
		Examples:    []string{`m = {'a': 1}; map_del(m, 'a')  ->  true`},
	},
	"map_clear": {
		Params:      []Param{p("m", "map")},
		Returns:     "unit",
		Description: "Removes every entry from m in place.",
		Examples:    []string{`map_clear(m); len(m)  ->  0`},
	},

	// ---- Arrays ----

	"indexes_to_keys": {
		Params:      []Param{p("arr", "array"), p("obj", "map")},
		Returns:     "map",
		Description: "Builds a new map by picking elements out of arr at the indexes named in obj. Every key of obj must be an int (an index into arr, in range) and every value of obj must be a valid map key itself (an int or a string) -- that value becomes the key in the result, mapped to arr's element at that index. Does not mutate arr.",
		Examples: []string{
			`indexes_to_keys(['x', 'y', 'z'], {0: 'first', 2: 'third'})  ->  {'first': 'x', 'third': 'z'}`,
			`indexes_to_keys(['x', 'y', 'z'], {})  ->  {}`,
		},
	},
	"sort": {
		Params:      []Param{p("arr", "array"), p("cmp", "fn")},
		Returns:     "array",
		Description: "A new, sorted array holding the elements of arr. cmp must take exactly two arguments and return an int: negative if the first should sort before the second, positive if after, zero if they're equal. Does not mutate arr.",
		Examples: []string{
			`sort([3, 1, 2], \(a, b) -> a - b)  ->  [1, 2, 3]`,
			`sort([3, 1, 2], \(a, b) -> b - a)  ->  [3, 2, 1]`,
		},
	},
	"concat": {
		Params:      []Param{pv("arrays", "array")},
		Returns:     "array",
		Description: "A new array holding every element of every argument array, in order. Called with no arguments, returns an empty array. Does not mutate any of the source arrays.",
		Examples:    []string{`concat([1, 2], [3, 4])  ->  [1, 2, 3, 4]`},
	},
	"contains": {
		Params:      []Param{p("target", "string|array|map"), p("needle", "any")},
		Returns:     "bool",
		Description: "For a string target, whether needle (a string) is a substring. For an array, whether any element equals needle (by value, with int/float compared numerically). For a map, whether needle (an int or string) exists as a key.",
		Examples: []string{
			`contains('Hello World', 'World')  ->  true`,
			`contains([1, 2, 3], 2)  ->  true`,
		},
	},
	"map": {
		Params:      []Param{p("arr", "array"), p("fn", "fn")},
		Returns:     "array",
		Description: "A new array with fn applied to every element of arr, in order. fn must take exactly one argument. Does not mutate arr.",
		Examples:    []string{`map([1, 2, 3], \(x) -> x * 2)  ->  [2, 4, 6]`},
	},
	"filter": {
		Params:      []Param{p("arr", "array"), p("fn", "fn")},
		Returns:     "array",
		Description: "A new array holding only the elements of arr for which fn returns a truthy value. fn must take exactly one argument. Does not mutate arr.",
		Examples:    []string{`filter([1, 2, 3, 4, 5], \(x) -> x > 2)  ->  [3, 4, 5]`},
	},
	"append": {
		Params:      []Param{p("arr", "array"), p("value", "any")},
		Returns:     "array",
		Description: "Appends value to the end of arr in place, and returns arr.",
		Examples:    []string{`a = [1]; append(a, 2); a  ->  [1, 2]`},
	},
	"delete": {
		Params:      []Param{p("arr", "array"), p("start", "int"), p("end", "int")},
		Returns:     "array",
		Description: "Removes elements from arr in place and returns arr. With just start, removes that single index. With end too, removes the half-open range [start, end) -- start through end-1. Throws a runtime error if start or end is out of range, or start > end.",
		Examples: []string{
			`a = [1, 2, 3, 4, 5]; delete(a, 2); a  ->  [1, 2, 4, 5]`,
			`a = [1, 2, 3, 4, 5]; delete(a, 1, 4); a  ->  [1, 5]`,
		},
	},
	"reverse": {
		Params:      []Param{p("arr", "array")},
		Returns:     "array",
		Description: "A new array with the same values of the original but in the reverse order. Does not mutate arr.",
		Examples:    []string{`a = [1, 2, 3, 4, 5]; reverse(a)  -> [5, 4, 3, 2, 1]`},
	},

	// ---- Random ----

	"srand": {
		Params:      []Param{p("seed", "int")},
		Returns:     "unit",
		Description: "Seeds the random number generator (a glibc-compatible rand()/srand() implementation). This is process-global state, shared even across import()ed modules.",
		Examples:    []string{`srand(4)`},
	},
	"rand": {
		Params:      []Param{p("min", "int"), p("max", "int")},
		Returns:     "int",
		Description: "A pseudo-random integer in the inclusive range [min, max]. Throws a runtime error if min > max.",
		Examples:    []string{`srand(4); rand(1, 10)  ->  2`},
	},

	// ---- Date & Time ----

	"time": {
		Returns:     "int",
		Description: "Current Unix time, in whole seconds.",
		Examples:    []string{`time()`},
	},
	"year":   dtDoc("year", "UTC year", "int"),
	"month":  dtDoc("month", "UTC month (1-12)", "int"),
	"date":   dtDoc("date", "UTC day of the month (1-31)", "int"),
	"hour":   dtDoc("hour", "UTC hour (0-23)", "int"),
	"minute": dtDoc("minute", "UTC minute (0-59)", "int"),
	"second": dtDoc("second", "UTC second (0-59)", "int"),
	"day": {
		Params:      []Param{p("offset_hours", "int")},
		Returns:     "int",
		Description: "Day of the week, UTC, as of (now + offset_hours hours): 0 = Sunday .. 6 = Saturday. Pass 0 for the current day of the week.",
		Examples:    []string{`day(0)  -- today's weekday, 0-6`},
	},
	"millisecond": {
		Returns:     "int",
		Description: "Current Unix time, in whole milliseconds. Unlike year/month/date/day/hour/minute/second, this takes no offset argument.",
		Examples:    []string{`millisecond()`},
	},
}

// mathDoc builds the shared doc shape for the plain float64->float64 math
// builtins (sin, cos, sqrt, ...): one numeric argument, numeric return.
func mathDoc(name, description string) Doc {
	return Doc{
		Params:      []Param{p("x", "int|float|bool")},
		Returns:     "int|float",
		Description: description,
		Examples:    []string{name + "(0)"},
	}
}

// isDoc builds the shared doc shape for the is_<kind> family.
func isDoc(kind string) Doc {
	return Doc{
		Params:      []Param{p("value", "any")},
		Returns:     "bool",
		Description: "True if value's kind is " + kind + ".",
		Examples:    []string{"is_" + kind + "(...)"},
	}
}

// dtDoc builds the shared doc shape for the hour-offset datetime builtins
// (year, month, date, hour, minute, second).
func dtDoc(name, field, returns string) Doc {
	return Doc{
		Params:      []Param{p("offset_hours", "int")},
		Returns:     returns,
		Description: field + ", as of (now + offset_hours hours). Pass 0 for the current value.",
		Examples:    []string{name + "(0)  -- current " + field},
	}
}

// builtinHelp documents a builtin named either by the function itself --
// help(sort), which now evaluates to the builtin's value like any other
// identifier -- or by its name, help('sort').
func builtinHelp(in *Interp, c *Call) Value {
	if c.N() > 1 {
		throw(c.Site, "too many arguments help(...). expected 0 or 1 but got %d", c.N())
	}

	if c.N() == 0 {
		printHelpOverview(in)
		return UnitV()
	}

	var name string

	switch v := c.Args[0].(type) {
	case StringValue:
		name = string(v)
	case *FnValue:
		if v.Native == nil {
			throw(c.At(0), "there is no help for a function you wrote yourself -- help only documents builtins")
		}
		name = v.Native.Name
	default:
		throw(c.At(0), "expected a builtin function or its name as a string, but got a '%s'", v.Kind())
	}

	// A package's own name documents the package; a qualified name documents
	// one of its functions. Both are checked before helpDocs, which only ever
	// holds the always-there builtins.
	if m, ok := nativeModules[name]; ok {
		printModuleOverview(in, m)
		return UnitV()
	}

	if modName, fnName, ok := splitQualified(name); ok {
		m, found := nativeModules[modName]
		if !found {
			throw(c.At(0), "there is no package named '%s'", modName)
		}

		doc, found := m.Docs[fnName]
		if !found {
			throw(c.At(0), "package '%s' has no function named '%s' -- run help('%s') to list what it does have", modName, fnName, modName)
		}

		printHelpEntry(in, name, doc)
		return UnitV()
	}

	doc, ok := helpDocs[name]
	if !ok {
		throw(c.At(0), "no help available for '%s' -- run help() to list all builtin functions", name)
	}

	printHelpEntry(in, name, doc)

	return UnitV()
}

// printModuleOverview lists one package's functions, the way
// printHelpOverview lists the builtin categories.
func printModuleOverview(in *Interp, m *Module) {
	names := make([]string, 0, len(m.Fns))
	for name := range m.Fns {
		names = append(names, name)
	}
	sort.Strings(names)

	writeOut(in, "package '"+m.Name+"' -- const "+m.Name+" = import('"+m.Name+"')\n\n")

	for _, name := range names {
		qualified := m.Name + "." + name

		if doc, ok := m.Docs[name]; ok {
			writeOut(in, "  "+helpSignature(qualified, doc)+"\n")
		} else {
			writeOut(in, "  "+qualified+"\n")
		}
	}

	writeOut(in, "\nRun help('"+m.Name+".<name>') for details on one of them.\n")
}

func helpSignature(name string, d Doc) string {
	parts := make([]string, len(d.Params))
	for i, p := range d.Params {
		n := p.Name
		if p.Variadic {
			n += "..."
		}
		parts[i] = n + ": " + p.Type
	}
	return name + "(" + strings.Join(parts, ", ") + ") -> " + d.Returns
}

func printHelpEntry(in *Interp, name string, d Doc) {
	writeOut(in, helpSignature(name, d))
	writeOut(in, "\n\n  ")
	writeOut(in, d.Description)
	writeOut(in, "\n")

	if len(d.Examples) > 0 {
		writeOut(in, "\nExamples:\n")
		for _, ex := range d.Examples {
			writeOut(in, "  ")
			writeOut(in, ex)
			writeOut(in, "\n")
		}
	}
}

func printHelpOverview(in *Interp) {
	writeOut(in, "MCA builtin functions -- run help('name') for details on a specific one.\n")

	for _, cat := range helpCategories {
		names := append([]string(nil), cat.Funcs...)
		sort.Strings(names)

		writeOut(in, "\n")
		writeOut(in, cat.Name)
		writeOut(in, ":\n  ")
		writeOut(in, strings.Join(names, ", "))
		writeOut(in, "\n")
	}

	// Packages are not builtins: they are not bound to any name until the
	// program imports one, so they are listed apart from the categories above.
	if len(nativeModules) == 0 {
		return
	}

	names := make([]string, 0, len(nativeModules))
	for name := range nativeModules {
		names = append(names, name)
	}
	sort.Strings(names)

	writeOut(in, "\nPackages (import to use, e.g. const c = import('"+names[0]+"')):\n  ")
	writeOut(in, strings.Join(names, ", "))
	writeOut(in, "\n")
}
