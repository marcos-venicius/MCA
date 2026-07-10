package interp

import "mca/internal/ast"

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
