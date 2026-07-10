package interp

import "mca/internal/ast"

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
