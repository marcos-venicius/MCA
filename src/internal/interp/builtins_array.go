package interp

import (
	"mca/internal/ast"
	"slices"
	"strings"
)

func builtinIndexesToKeys(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	array := expectKind(args[0], in.Eval(args[0]).Value, KArray).(*Array)
	obj := expectKind(args[1], in.Eval(args[1]).Value, KMap).(*Map)

	out := make(map[MapKey]Value)

	for k, v := range obj.values {
		if k.Kind != KInt {
			throw(args[1].Pos(), "'%s' is not an integer", k.String())
		}

		idx := int(k.I)

		if idx < 0 || idx >= len(array.Items) {
			throw(args[1].Pos(), "index %d is out of range. array has %d elements", idx, len(array.Items))
		}

		if !isValidMapKeyType(v.Kind()) {
			throw(args[1].Pos(), "%s is not a valid map key data type", v.Kind())
		}

		mk, _ := mapKeyFromValue(v)

		out[mk] = array.Items[idx]
	}

	return MapV(&Map{values: out})
}

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

// builtinDelete implements delete(array, start, end?): removes the single
// index start, or the range [start, end) when end is given -- same
// half-open convention as select()'s [from, to). Mutates array in place and
// returns it.
func builtinDelete(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	if len(args) > 3 {
		throw(caller.Pos(), "too many arguments delete(...). expected 2 or 3 but got %d", len(args))
	} else if len(args) < 2 {
		throw(caller.Pos(), "too few arguments delete(...). expected 2 or 3 but got %d", len(args))
	}

	arr := expectKind(args[0], in.Eval(args[0]).Value, KArray).(*Array)
	start := intOf(expectKind(args[1], in.Eval(args[1]).Value, KInt))

	length := int64(len(arr.Items))

	end := start + 1
	if len(args) == 3 {
		end = intOf(expectKind(args[2], in.Eval(args[2]).Value, KInt))
	}

	if start < 0 || start >= length {
		throw(args[1].Pos(), "start '%d' is out of range. The size of the array is %d", start, length)
	}
	if end < 0 || end > length {
		throw(args[len(args)-1].Pos(), "end '%d' is out of range. The size of the array is %d", end, length)
	}
	if start > end {
		throw(args[1].Pos(), "start '%d' cannot be greater than end '%d'", start, end)
	}

	arr.Items = append(arr.Items[:start], arr.Items[end:]...)

	return ArrayV(arr)
}
