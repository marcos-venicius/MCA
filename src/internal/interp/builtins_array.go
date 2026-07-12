package interp

import (
	"mca/internal/ast"
	"slices"
	"strings"
)

func builtinSort(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	array := expectKind(args[0], in.Eval(args[0]).Value, KArray).(*Array)
	lambda := expectKind(args[1], in.Eval(args[1]).Value, KFn).(*FnValue)

	if len(lambda.Node.Params) != 2 {
		throw(args[1].Pos(), "expected two arguments but got %d", len(lambda.Node.Params))
	}

	copy := slices.Clone(array.Items)

	slices.SortFunc(copy, func(a, b Value) int {
		result := in.callFnValue(lambda, args[1].Pos(), calleeLabel(nil), []Value{a, b})

		if result.Kind() != KInt {
			throw(args[1].Pos(), "the sorting function should return an integer but returned %s. try `help(sort)`", result.Kind())
		}

		return int(result.(IntValue))
	})

	return ArrayV(&Array{
		Items: copy,
	})
}

func builtinReverse(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	value := expectKind(args[0], in.Eval(args[0]).Value, KArray).(*Array)

	out := make([]Value, len(value.Items))

	for i := range value.Items {
		out[i] = value.Items[len(value.Items)-i-1]
	}

	arr := Array{
		Items: out,
	}

	return ArrayV(&arr)
}

func builtinConcat(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	out := make([]Value, 0)

	for _, arg := range args {
		value := expectKind(arg, in.Eval(arg).Value, KArray).(*Array)

		out = append(out, value.Items...)
	}

	arr := Array{
		Items: out,
	}

	return ArrayV(&arr)
}

func builtinContains(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	target := expectKind(args[0], in.Eval(args[0]).Value, KString, KArray, KMap)

	switch target.Kind() {
	case KString:
		substr := string(expectKind(args[1], in.Eval(args[1]).Value, KString).(StringValue))

		return BoolV(strings.Contains(string(target.(StringValue)), substr))
	case KArray:
		value := in.Eval(args[1]).Value

		items := (target.(*Array)).Items

		for _, v := range items {
			if compareTwoValues(v, value) {
				return BoolV(true)
			}
		}

		return BoolV(false)
	case KMap:
		key := expectKind(args[1], in.Eval(args[1]).Value, KString, KInt)

		mk, _ := mapKeyFromValue(key)
		m := (target.(*Map)).values

		if _, ok := m[mk]; ok {
			return BoolV(true)
		}

		return BoolV(false)
	}

	panic("builtinContains: unreacheable")
}

func builtinFilter(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	arr := expectKind(args[0], in.Eval(args[0]).Value, KArray).(*Array).Items
	fn := expectKind(args[1], in.Eval(args[1]).Value, KFn).(*FnValue)

	if len(fn.Node.Params) != 1 {
		throw(args[1].Pos(), "filter closure should expect exactly one argument, but it has %d", len(fn.Node.Params))
	}

	out := make([]Value, 0, len(arr))

	for i, v := range arr {
		isTruthy, ok := Truthy(in.callFnValue(fn, caller.Pos(), calleeLabel(nil), []Value{v}))

		if !ok {
			throw(args[1].Pos(), "failed when applying closure to array value at index %d of type %s. the closure didn't returned a truthy value", i, v.Kind())
		}

		if isTruthy {
			out = append(out, v)
		}
	}

	filtered := Array{
		Items: out,
	}

	return &filtered
}

func builtinMap(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	arr := expectKind(args[0], in.Eval(args[0]).Value, KArray).(*Array).Items
	fn := expectKind(args[1], in.Eval(args[1]).Value, KFn).(*FnValue)

	if len(fn.Node.Params) != 1 {
		throw(args[1].Pos(), "map closure should expect exactly one argument, but it has %d", len(fn.Node.Params))
	}

	out := make([]Value, len(arr))

	for i, v := range arr {
		value := in.callFnValue(fn, caller.Pos(), calleeLabel(nil), []Value{v})

		out[i] = value
	}

	mapped := Array{
		Items: out,
	}

	return &mapped
}

func builtinAppend(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	arrVal := in.Eval(args[0]).Value

	arr, ok := arrVal.(*Array)
	if !ok {
		throw(args[0].Pos(), "first argument to append must be an array")
	}

	val := in.Eval(args[1]).Value
	arr.Items = append(arr.Items, val)

	return arrVal
}
