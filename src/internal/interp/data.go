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

func mapValueFromKey(v MapKey) Value {
	switch v.Kind {
	case KString:
		return StringV(v.S)
	case KInt:
		return IntV(v.I)
	}

	panic("mapValueFromKey: unreacheable")
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
		var keyVal Value

		if ident, ok := e.Keys[i].(*ast.Ident); ok {
			keyVal = StringV(ident.Name)
		} else {
			keyVal = expectKind(e.Keys[i], in.Eval(e.Keys[i]).Value, KInt, KString)
		}
		mk, _ := mapKeyFromValue(keyVal)

		valVal := expectKind(e.Values[i], in.Eval(e.Values[i]).Value, KString, KInt, KBool, KFloat, KFn, KMap, KArray, KUnit)

		m.Set(mk, valVal)
	}

	return normal(MapV(m))
}

func (in *Interp) evalSquare(e *ast.SquareExpr) EvalResult {
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
		idxVal := expectKind(e.Index, in.Eval(e.Index).Value, KInt, KString)
		mk, _ := mapKeyFromValue(idxVal)
		if v, ok := lv.Get(mk); ok {
			return normal(v)
		}

		switch mk.Kind {
		case KInt:
			throw(e.Pos(), "key '%d' not found", mk.I)
		case KString:
			throw(e.Pos(), "key '%s' not found", mk.S)
		}
	}

	panic("evalSquare: unreachable")
}

func (in *Interp) evalDot(e *ast.DotExpr) EvalResult {
	left := expectKind(e.Left, in.Eval(e.Left).Value, KMap)

	lv := left.(*Map)

	// The parser only ever produces an Ident here (m.f(...) parses as
	// CallExpr{Callee: DotExpr{Index: Ident("f")}}; the call itself is
	// handled generically by evalCall, not here).
	ident := e.Index.(*ast.Ident)

	if v, ok := lv.Get(MapKey{Kind: KString, S: ident.Name}); ok {
		return normal(v)
	}

	throw(e.Pos(), "key '%s' not found", ident.Name)
	panic("evalDot: unreacheable")
}

// storeSquareAssign handles array/map index-assignment targets (`arr[i] = v`,
// `m[k] = v`). It only performs the store -- the assignment
// expression's own result (including Flow) is the right-hand side's
// EvalResult, handled by the caller (evalAssign).
func (in *Interp) storeSquareAssign(e *ast.AssignExpr, left *ast.SquareExpr, rightVal Value) {
	leftVal := expectKind(left.Left, in.Eval(left.Left).Value, KArray, KMap)

	switch lv := leftVal.(type) {
	case *Map:
		expectKind(e.Right, rightVal, KString, KInt, KBool, KFloat, KFn, KMap, KArray, KUnit)

		idxVal := expectKind(left.Index, in.Eval(left.Index).Value, KInt, KString)
		mk, _ := mapKeyFromValue(idxVal)

		lv.Set(mk, rightVal)

	case *Array:
		idxVal := expectKind(left.Index, in.Eval(left.Index).Value, KInt)
		i := intOf(idxVal)

		if i < 0 || i >= int64(len(lv.Items)) {
			throw(e.Pos(), "array index out of bounds")
		}

		lv.Items[i] = rightVal

	default:
		panic("storeSquareAssign: unreachable")
	}
}

// storeDotAssign handles map index-assignment targets
// (`m.k = v`). It only performs the store -- the assignment
// expression's own result (including Flow) is the right-hand side's
// EvalResult, handled by the caller (evalAssign).
func (in *Interp) storeDotAssign(e *ast.AssignExpr, left *ast.DotExpr, rightVal Value) {
	leftVal := expectKind(left.Left, in.Eval(left.Left).Value, KMap)

	lv := leftVal.(*Map)

	expectKind(e.Right, rightVal, KString, KInt, KBool, KFloat, KFn, KMap, KArray, KUnit)

	switch node := left.Index.(type) {
	case *ast.Ident:
		// `m.field` sugar: always a literal string key, never a variable lookup.
		lv.Set(MapKey{Kind: KString, S: node.Name}, rightVal)
	default:
		throw(left.Index.Pos(), "invalid use of dot operator. only accept valid identifiers in assignments")
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
