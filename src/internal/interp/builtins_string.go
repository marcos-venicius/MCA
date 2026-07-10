package interp

import (
	"strconv"
	"strings"

	"mca/internal/ast"
)

func builtinLen(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	v := expectKind(args[0], in.Eval(args[0]).Value, KString, KMap, KArray)

	switch vv := v.(type) {
	case StringValue:
		return IntV(int64(len(vv)))
	case *Map:
		return IntV(int64(vv.Len()))
	default: // *Array
		return IntV(int64(len(vv.(*Array).Items)))
	}
}

func builtinJoin(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	arr := expectKind(args[0], in.Eval(args[0]).Value, KArray).(*Array)
	sep := expectKind(args[1], in.Eval(args[1]).Value, KString).(StringValue)

	strs := make([]string, len(arr.Items))

	for i, v := range arr.Items {
		if v.Kind() != KString {
			throw(args[0].Pos(), "expected a string at index %d but got '%s'", i, v.Kind())
		}

		strs[i] = string(v.(StringValue))
	}

	out := strings.Join(strs, string(sep))

	return StringV(out)
}

func builtinSplit(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	str := string(expectKind(args[0], in.Eval(args[0]).Value, KString).(StringValue))
	sep := string(expectKind(args[1], in.Eval(args[1]).Value, KString).(StringValue))

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
func builtinSelect(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	data := stringOf(expectKind(args[0], in.Eval(args[0]).Value, KString))
	from := intOf(expectKind(args[1], in.Eval(args[1]).Value, KInt))
	to := intOf(expectKind(args[2], in.Eval(args[2]).Value, KInt))

	length := int64(len(data))

	if from < 0 || from >= length {
		throw(args[1].Pos(), "from '%d' is out of range. The size of the string is %d", from, length)
	}
	if to < 0 || to >= length+1 {
		throw(args[2].Pos(), "to '%d' is out of range. The size of the string is %d", to, length)
	}
	if from > to {
		throw(args[1].Pos(), "from '%d' cannot be greater than to '%d'", from, to)
	}

	return StringV(data[from:to])
}

func builtinOrd(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	data := stringOf(expectKind(args[0], in.Eval(args[0]).Value, KString))

	if len(data) != 1 {
		throw(args[0].Pos(), "ord() expects a string of length 1, got a string of length %d", len(data))
	}

	return IntV(int64(data[0]))
}

func builtinChr(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	data := intOf(expectKind(args[0], in.Eval(args[0]).Value, KInt))

	return StringV(string(rune(data)))
}

func builtinFormat(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	if len(args) <= 0 {
		throw(caller.Pos(), "expected at least one argument but received %d", len(args))
	}

	var sb strings.Builder

	for _, arg := range args {
		v := expectKind(arg, in.Eval(arg).Value, KInt, KString, KFloat, KBool)

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
