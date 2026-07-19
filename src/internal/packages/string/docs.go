package string

import "mca/internal/interp"

func p(name, typ string) interp.Param { return interp.Param{Name: name, Type: typ} }

var docs = map[string]interp.Doc{
	"repeat": {
		Params:      []interp.Param{p("str", "string"), p("count", "int")},
		Returns:     "string",
		Description: "str repeated count times, concatenated with no separator. Throws a runtime error if count is negative.",
		Examples:    []string{`string.repeat('ab', 3)  ->  'ababab'`},
	},
	"replace": {
		Params:      []interp.Param{p("str", "string"), p("old", "string"), p("new", "string")},
		Returns:     "string",
		Description: "str with every non-overlapping occurrence of old replaced by new.",
		Examples: []string{
			`string.replace('Hello World', 'World', 'There')  ->  'Hello There'`,
			`string.replace('aaa', 'a', 'b')  ->  'bbb'`,
		},
	},
	"starts_with": {
		Params:      []interp.Param{p("str", "string"), p("prefix", "string")},
		Returns:     "bool",
		Description: "True if str starts with prefix. Case-sensitive; an empty prefix always matches.",
		Examples:    []string{`string.starts_with('Hello World', 'Hello')  ->  true`},
	},
	"ends_with": {
		Params:      []interp.Param{p("str", "string"), p("suffix", "string")},
		Returns:     "bool",
		Description: "True if str ends with suffix. Case-sensitive; an empty suffix always matches.",
		Examples:    []string{`string.ends_with('Hello World', 'World')  ->  true`},
	},
	"lower": {
		Params:      []interp.Param{p("str", "string")},
		Returns:     "string",
		Description: "str with every letter lowercased.",
		Examples:    []string{`string.lower('HELLO')  ->  'hello'`},
	},
	"upper": {
		Params:      []interp.Param{p("str", "string")},
		Returns:     "string",
		Description: "str with every letter uppercased.",
		Examples:    []string{`string.upper('hello')  ->  'HELLO'`},
	},
	"trim": {
		Params:      []interp.Param{p("str", "string")},
		Returns:     "string",
		Description: "str with leading and trailing whitespace removed.",
		Examples:    []string{`string.trim('  hello  ')  ->  'hello'`},
	},
	"ltrim": {
		Params:      []interp.Param{p("str", "string")},
		Returns:     "string",
		Description: "str with leading whitespace removed.",
		Examples:    []string{`string.ltrim('  hello  ')  ->  'hello  '`},
	},
	"rtrim": {
		Params:      []interp.Param{p("str", "string")},
		Returns:     "string",
		Description: "str with trailing whitespace removed.",
		Examples:    []string{`string.rtrim('  hello  ')  ->  '  hello'`},
	},
	"join": {
		Params:      []interp.Param{p("items", "array"), p("sep", "string")},
		Returns:     "string",
		Description: "Concatenates items (every element must be a string) with sep placed between each pair.",
		Examples:    []string{`string.join(['a', 'b', 'c'], ',')  ->  'a,b,c'`},
	},
	"split": {
		Params:      []interp.Param{p("str", "string"), p("sep", "string")},
		Returns:     "array",
		Description: "Splits str on every occurrence of sep into an array of strings. If sep doesn't occur, returns a single-element array holding the whole string.",
		Examples:    []string{`string.split('a,b,c', ',')  ->  ['a', 'b', 'c']`},
	},
	"ord": {
		Params:      []interp.Param{p("char", "string")},
		Returns:     "int",
		Description: "Byte value of char, which must be a string of length exactly 1.",
		Examples:    []string{`string.ord('a')  ->  97`},
	},
	"chr": {
		Params:      []interp.Param{p("codepoint", "int")},
		Returns:     "string",
		Description: "The UTF-8 string for the Unicode codepoint. Codepoints outside the valid range fall back to the replacement character (U+FFFD) rather than erroring.",
		Examples:    []string{`string.chr(65)  ->  'A'`},
	},
	"format": {
		Params:      []interp.Param{{Name: "values", Type: "int|string|float|bool", Variadic: true}},
		Returns:     "string",
		Description: "Concatenates one or more values into a single string with no separator (ints/floats/bools are stringified). Requires at least one argument.",
		Examples:    []string{`string.format('I am ', 5, ' years old')  ->  'I am 5 years old'`},
	},
}
