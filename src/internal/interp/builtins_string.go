package interp

import (
	"strconv"
	"strings"
	"unicode"
)

func builtinLen(in *Interp, c *Call) Value {
	v := expectKindAt(c.At(0), c.Args[0], KString, KMap, KArray)

	switch vv := v.(type) {
	case StringValue:
		return IntV(int64(len(vv)))
	case *Map:
		return IntV(int64(vv.Len()))
	default: // *Array
		return IntV(int64(len(vv.(*Array).Items)))
	}
}

func builtinRepeat(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))
	n := int(expectKindAt(c.At(1), c.Args[1], KInt).(IntValue))

	if n < 0 {
		throw(c.At(1), "repeat count cannot be negative, got %d", n)
	}

	return StringV(strings.Repeat(str, n))
}

func builtinReplace(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))
	old := string(expectKindAt(c.At(1), c.Args[1], KString).(StringValue))
	new := string(expectKindAt(c.At(2), c.Args[2], KString).(StringValue))

	return StringV(strings.ReplaceAll(str, old, new))
}

func builtinStartsWith(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))
	prefix := string(expectKindAt(c.At(1), c.Args[1], KString).(StringValue))

	return BoolV(strings.HasPrefix(str, prefix))
}

func builtinEndsWith(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))
	suffix := string(expectKindAt(c.At(1), c.Args[1], KString).(StringValue))

	return BoolV(strings.HasSuffix(str, suffix))
}

func builtinLower(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))

	lower := strings.ToLower(str)

	return StringV(lower)
}

func builtinUpper(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))

	upper := strings.ToUpper(str)

	return StringV(upper)
}

func builtinTrim(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))

	out := strings.TrimSpace(str)

	return StringV(out)
}

func builtinLTrim(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))

	out := strings.TrimLeftFunc(str, unicode.IsSpace)

	return StringV(out)
}

func builtinRTrim(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))

	out := strings.TrimRightFunc(str, unicode.IsSpace)

	return StringV(out)
}

func builtinJoin(in *Interp, c *Call) Value {
	arr := expectKindAt(c.At(0), c.Args[0], KArray).(*Array)
	sep := expectKindAt(c.At(1), c.Args[1], KString).(StringValue)

	strs := make([]string, len(arr.Items))

	for i, v := range arr.Items {
		if v.Kind() != KString {
			throw(c.At(0), "expected a string at index %d but got '%s'", i, v.Kind())
		}

		strs[i] = string(v.(StringValue))
	}

	out := strings.Join(strs, string(sep))

	return StringV(out)
}

func builtinSplit(in *Interp, c *Call) Value {
	str := string(expectKindAt(c.At(0), c.Args[0], KString).(StringValue))
	sep := string(expectKindAt(c.At(1), c.Args[1], KString).(StringValue))

	out := strings.Split(str, sep)

	arr := Array{
		Items: make([]Value, len(out)),
	}

	for i, v := range out {
		arr.Items[i] = StringV(v)
	}

	return ArrayV(&arr)
}

// TODO: later, instead of a builtin function I want to make it a 'range operator'
// just like in python 'Hello'[1:3]
func builtinSelect(in *Interp, c *Call) Value {
	data := stringOf(expectKindAt(c.At(0), c.Args[0], KString))
	from := intOf(expectKindAt(c.At(1), c.Args[1], KInt))
	to := intOf(expectKindAt(c.At(2), c.Args[2], KInt))

	length := int64(len(data))

	if from < 0 || from >= length {
		throw(c.At(1), "from '%d' is out of range. The size of the string is %d", from, length)
	}
	if to < 0 || to >= length+1 {
		throw(c.At(2), "to '%d' is out of range. The size of the string is %d", to, length)
	}
	if from > to {
		throw(c.At(1), "from '%d' cannot be greater than to '%d'", from, to)
	}

	return StringV(data[from:to])
}

func builtinOrd(in *Interp, c *Call) Value {
	data := stringOf(expectKindAt(c.At(0), c.Args[0], KString))

	if len(data) != 1 {
		throw(c.At(0), "ord() expects a string of length 1, got a string of length %d", len(data))
	}

	return IntV(int64(data[0]))
}

func builtinChr(in *Interp, c *Call) Value {
	data := intOf(expectKindAt(c.At(0), c.Args[0], KInt))

	return StringV(string(rune(data)))
}

func builtinFormat(in *Interp, c *Call) Value {
	if c.N() <= 0 {
		throw(c.Site, "expected at least one argument but received %d", c.N())
	}

	var sb strings.Builder

	for i, arg := range c.Args {
		v := expectKindAt(c.At(i), arg, KInt, KString, KFloat, KBool)

		switch vv := v.(type) {
		case IntValue:
			sb.WriteString(strconv.FormatInt(int64(vv), 10))
		case FloatValue:
			sb.WriteString(strconv.FormatFloat(float64(vv), 'g', 6, 64))
		case BoolValue:
			if vv {
				sb.WriteString("true")
			} else {
				sb.WriteString("false")
			}
		case StringValue:
			sb.WriteString(string(vv))
		}
	}

	return StringV(sb.String())
}
