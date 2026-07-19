// Package string is MCA's `string` package: the text-manipulation functions
// that used to be always-bound builtins, reached with
// `const string = import('string')`. Behavior is unchanged from the builtin
// era -- indices and lengths are byte-based, not rune-based.
//
// ('string' is a predeclared identifier in Go, not a keyword, so it is a
// legal package name; the predeclared type stays reachable inside these files
// because a package's own name is never in scope within itself.)
package string

import (
	"strconv"
	"strings"
	"unicode"

	"mca/internal/interp"
)

func init() {
	interp.RegisterModule(&interp.Module{
		Name: "string",
		Fns: map[string]*interp.Native{
			"repeat":      interp.NewNative("string.repeat", 2, repeat),
			"replace":     interp.NewNative("string.replace", 3, replace),
			"starts_with": interp.NewNative("string.starts_with", 2, startsWith),
			"ends_with":   interp.NewNative("string.ends_with", 2, endsWith),
			"lower":       interp.NewNative("string.lower", 1, lower),
			"upper":       interp.NewNative("string.upper", 1, upper),
			"trim":        interp.NewNative("string.trim", 1, trim),
			"ltrim":       interp.NewNative("string.ltrim", 1, ltrim),
			"rtrim":       interp.NewNative("string.rtrim", 1, rtrim),
			"join":        interp.NewNative("string.join", 2, join),
			"split":       interp.NewNative("string.split", 2, split),
			"select":      interp.NewNative("string.select", 3, sel),
			"ord":         interp.NewNative("string.ord", 1, ord),
			"chr":         interp.NewNative("string.chr", 1, chr),
			"format":      interp.NewNative("string.format", -1, format),
		},
		Docs: docs,
	})
}

func repeat(in *interp.Interp, c *interp.Call) interp.Value {
	str := c.StringArg(0)
	n := c.IntArg(1)

	if n < 0 {
		interp.Throw(c.At(1), "repeat count cannot be negative, got %d", n)
	}

	return interp.StringV(strings.Repeat(str, int(n)))
}

func replace(in *interp.Interp, c *interp.Call) interp.Value {
	str := c.StringArg(0)
	old := c.StringArg(1)
	new := c.StringArg(2)

	return interp.StringV(strings.ReplaceAll(str, old, new))
}

func startsWith(in *interp.Interp, c *interp.Call) interp.Value {
	return interp.BoolV(strings.HasPrefix(c.StringArg(0), c.StringArg(1)))
}

func endsWith(in *interp.Interp, c *interp.Call) interp.Value {
	return interp.BoolV(strings.HasSuffix(c.StringArg(0), c.StringArg(1)))
}

func lower(in *interp.Interp, c *interp.Call) interp.Value {
	return interp.StringV(strings.ToLower(c.StringArg(0)))
}

func upper(in *interp.Interp, c *interp.Call) interp.Value {
	return interp.StringV(strings.ToUpper(c.StringArg(0)))
}

func trim(in *interp.Interp, c *interp.Call) interp.Value {
	return interp.StringV(strings.TrimSpace(c.StringArg(0)))
}

func ltrim(in *interp.Interp, c *interp.Call) interp.Value {
	return interp.StringV(strings.TrimLeftFunc(c.StringArg(0), unicode.IsSpace))
}

func rtrim(in *interp.Interp, c *interp.Call) interp.Value {
	return interp.StringV(strings.TrimRightFunc(c.StringArg(0), unicode.IsSpace))
}

func join(in *interp.Interp, c *interp.Call) interp.Value {
	arr := c.ArrayArg(0)
	sep := c.StringArg(1)

	strs := make([]string, len(arr.Items))

	for i, v := range arr.Items {
		if v.Kind() != interp.KString {
			interp.Throw(c.At(0), "expected a string at index %d but got '%s'", i, v.Kind())
		}

		strs[i] = interp.AsString(v)
	}

	return interp.StringV(strings.Join(strs, sep))
}

func split(in *interp.Interp, c *interp.Call) interp.Value {
	str := c.StringArg(0)
	sep := c.StringArg(1)

	out := strings.Split(str, sep)

	arr := interp.Array{
		Items: make([]interp.Value, len(out)),
	}

	for i, v := range out {
		arr.Items[i] = interp.StringV(v)
	}

	return interp.ArrayV(&arr)
}

// TODO: later, instead of a builtin function I want to make it a 'range operator'
// just like in python 'Hello'[1:3]
func sel(in *interp.Interp, c *interp.Call) interp.Value {
	data := c.StringArg(0)
	from := c.IntArg(1)
	to := c.IntArg(2)

	length := int64(len(data))

	if from < 0 || from >= length {
		interp.Throw(c.At(1), "from '%d' is out of range. The size of the string is %d", from, length)
	}
	if to < 0 || to >= length+1 {
		interp.Throw(c.At(2), "to '%d' is out of range. The size of the string is %d", to, length)
	}
	if from > to {
		interp.Throw(c.At(1), "from '%d' cannot be greater than to '%d'", from, to)
	}

	return interp.StringV(data[from:to])
}

func ord(in *interp.Interp, c *interp.Call) interp.Value {
	data := c.StringArg(0)

	if len(data) != 1 {
		interp.Throw(c.At(0), "ord() expects a string of length 1, got a string of length %d", len(data))
	}

	return interp.IntV(int64(data[0]))
}

func chr(in *interp.Interp, c *interp.Call) interp.Value {
	return interp.StringV(string(rune(c.IntArg(0))))
}

func format(in *interp.Interp, c *interp.Call) interp.Value {
	if c.N() <= 0 {
		interp.Throw(c.Site, "expected at least one argument but received %d", c.N())
	}

	var sb strings.Builder

	for i := range c.Args {
		v := c.Arg(i, interp.KInt, interp.KString, interp.KFloat, interp.KBool)

		switch v.Kind() {
		case interp.KInt:
			sb.WriteString(strconv.FormatInt(interp.AsInt(v), 10))
		case interp.KFloat:
			sb.WriteString(strconv.FormatFloat(interp.AsFloat(v), 'g', 6, 64))
		case interp.KBool:
			if interp.AsBool(v) {
				sb.WriteString("true")
			} else {
				sb.WriteString("false")
			}
		case interp.KString:
			sb.WriteString(interp.AsString(v))
		}
	}

	return interp.StringV(sb.String())
}
