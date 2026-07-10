package interp

import "mca/internal/ast"

// TODO: since we have GO's GC on our behalf, do we need to have a map clear?
//       What if we just assign an empty object to the map?

func builtinMapDel(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	m := expectKind(args[0], in.Eval(args[0]).Value, KMap).(*Map)
	key := expectKind(args[1], in.Eval(args[1]).Value, KInt, KString)

	mk, _ := mapKeyFromValue(key)

	return BoolV(m.Del(mk))
}

func builtinMapClear(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	m := expectKind(args[0], in.Eval(args[0]).Value, KMap).(*Map)
	m.Clear()
	return UnitV()
}
