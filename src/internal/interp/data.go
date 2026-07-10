package interp

import "mca/internal/ast"

func mapKeyFromValue(v Value) (MapKey, bool) {
	switch vv := v.(type) {
	case IntValue:
		return MapKey{Kind: KInt, I: int64(vv)}, true
	case StringValue:
		return MapKey{Kind: KString, S: string(vv)}, true
	}
	return MapKey{}, false
}

func (in *Interp) evalArrayLit(e *ast.ArrayExpr) EvalResult {
	items := make([]Value, len(e.Items))
	for i, item := range e.Items {
		// Like every other multi-operand construct in this interpreter
		// (binary operands, call arguments, ...), only .Value is kept here;
		// a break/return reached through an item expression is discarded.
		// Control flow only propagates through block statement sequencing
		// (evalBlock).
		items[i] = in.Eval(item).Value
	}
	return normal(ArrayV(&Array{Items: items}))
}

func (in *Interp) evalMapLit(e *ast.MapExpr) EvalResult {
	m := NewMap()

	for i := range e.Keys {
		keyVal := expectKind(e.Keys[i], in.Eval(e.Keys[i]).Value, KInt, KString)
		mk, _ := mapKeyFromValue(keyVal)

		valVal := expectKind(e.Values[i], in.Eval(e.Values[i]).Value, KString, KInt, KBool, KFloat, KFn)

		m.Set(mk, valVal)
	}

	return normal(MapV(m))
}

func (in *Interp) evalIndex(e *ast.IndexExpr) EvalResult {
	left := expectKind(e.Left, in.Eval(e.Left).Value, KArray, KString, KMap)

	switch lv := left.(type) {
	case *Array:
		idx := expectKind(e.Index, in.Eval(e.Index).Value, KInt)
		i := intOf(idx)
		if i < 0 || i >= int64(len(lv.Items)) {
			throw(e.Pos(), "array index out of bounds")
		}
		return normal(lv.Items[i])

	case StringValue:
		idx := expectKind(e.Index, in.Eval(e.Index).Value, KInt)
		i := intOf(idx)
		if i < 0 || i >= int64(len(lv)) {
			throw(e.Pos(), "string index out of bounds")
		}
		return normal(StringV(string(lv[i])))

	case *Map:
		switch idxNode := e.Index.(type) {
		case *ast.CallExpr:
			// `m.method(args)` sugar: look the name up as a map entry and
			// invoke it directly, bypassing normal function/variable
			// resolution entirely. TODO: should I improve this somehow?
			entryVal := UnitV()
			if v, ok := lv.Get(MapKey{Kind: KString, S: idxNode.FnName}); ok {
				entryVal = v
			}
			fnVal := expectKind(idxNode, entryVal, KFn)
			return normal(in.callFn(fnVal.(*FnValue), idxNode.Pos(), idxNode.FnName, idxNode.Args))

		case *ast.Ident:
			// `m.field` sugar: always a literal string key, never a variable lookup.
			if v, ok := lv.Get(MapKey{Kind: KString, S: idxNode.Name}); ok {
				return normal(v)
			}
			return normal(UnitV()) // missing map key -> unit, not an error

		default:
			idxVal := expectKind(e.Index, in.Eval(e.Index).Value, KInt, KString)
			mk, _ := mapKeyFromValue(idxVal)
			if v, ok := lv.Get(mk); ok {
				return normal(v)
			}
			return normal(UnitV())
		}
	}

	panic("evalIndex: unreachable")
}

// storeIndexAssign handles array/map index-assignment targets (`arr[i] = v`,
// `m[k] = v`, `m.field = v`). It only performs the store -- the assignment
// expression's own result (including Flow) is the right-hand side's
// EvalResult, handled by the caller (evalAssign).
func (in *Interp) storeIndexAssign(e *ast.AssignExpr, left *ast.IndexExpr, rightVal Value) {
	leftVal := expectKind(left.Left, in.Eval(left.Left).Value, KArray, KMap)

	switch lv := leftVal.(type) {
	case *Map:
		expectKind(e.Right, rightVal, KString, KInt, KBool, KFloat, KFn)

		var mk MapKey
		if idNode, ok := left.Index.(*ast.Ident); ok {
			mk = MapKey{Kind: KString, S: idNode.Name}
		} else {
			idxVal := expectKind(left.Index, in.Eval(left.Index).Value, KInt, KString)
			mk, _ = mapKeyFromValue(idxVal)
		}

		lv.Set(mk, rightVal)

	case *Array:
		idxVal := expectKind(left.Index, in.Eval(left.Index).Value, KInt)
		i := intOf(idxVal)

		if i < 0 || i >= int64(len(lv.Items)) {
			throw(e.Pos(), "array index out of bounds")
		}

		lv.Items[i] = rightVal

	default:
		panic("storeIndexAssign: unreachable")
	}
}

// ---- for-of ----

func (in *Interp) evalForOf(e *ast.ForOfExpr) EvalResult {
	target := expectKind(e.Target, in.Eval(e.Target).Value, KArray, KString, KMap)

	switch t := target.(type) {
	case *Map:
		return in.forOfMap(e, t)
	case *Array:
		return in.forOfArray(e, t)
	case StringValue:
		return in.forOfString(e, string(t))
	}

	panic("evalForOf: unreachable")
}

// forOfLoopStep runs one iteration's body with key/value bound in a fresh
// scope. break stops the loop (and yields its value) uniformly across all
// loop kinds, same as while/for-range.
func (in *Interp) forOfLoopStep(body []ast.Expr, keyName, valName string, key, value Value, last *EvalResult) (stop bool) {
	if body == nil {
		return false
	}

	parent := in.pushScope()
	in.Current.Define(keyName, key)
	in.Current.Define(valName, value)
	*last = in.evalBlock(body)
	in.popScope(parent)

	if last.Flow == FlowReturn {
		return true
	}
	if last.Flow == FlowBreak {
		*last = normal(last.Value)
		return true
	}

	return false
}

func (in *Interp) forOfMap(e *ast.ForOfExpr, m *Map) EvalResult {
	last := normal(UnitV())

	for k, v := range m.values {
		keyVal := StringV(k.S)

		if k.Kind == KInt {
			keyVal = IntV(k.I)
		}

		if in.forOfLoopStep(e.Body, e.Key.Name, e.Value.Name, keyVal, v, &last) {
			break
		}
	}

	return last
}

func (in *Interp) forOfArray(e *ast.ForOfExpr, arr *Array) EvalResult {
	last := normal(UnitV())

	for i, v := range arr.Items {
		if in.forOfLoopStep(e.Body, e.Key.Name, e.Value.Name, IntV(int64(i)), v, &last) {
			break
		}
	}

	return last
}

func (in *Interp) forOfString(e *ast.ForOfExpr, s string) EvalResult {
	last := normal(UnitV())

	for i := 0; i < len(s); i++ {
		if in.forOfLoopStep(e.Body, e.Key.Name, e.Value.Name, IntV(int64(i)), StringV(string(s[i])), &last) {
			break
		}
	}

	return last
}
