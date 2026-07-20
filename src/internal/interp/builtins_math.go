package interp

// The rest of the numeric functions (sin, sqrt, floor, ...) live in the
// 'math' native package (internal/packages/math); sum, max and min stay
// builtins because they are everyday list operations more than mathematics.

func builtinSum(in *Interp, c *Call) Value {
	arr := arrayOf(expectKindAt(c.At(0), c.Args[0], KArray))

	hasFloat := false

	var r float64 = 0

	for _, v := range arr.Items {
		if v.Kind() != KInt && v.Kind() != KFloat {
			throw(c.At(0), "expected int | float but got %s", v.Kind())
		}

		if v.Kind() == KFloat {
			hasFloat = true

			r += floatOf(v)
		} else {
			r += float64(intOf(v))
		}
	}

	if hasFloat {
		return FloatV(r)
	}

	return IntV(int64(r))
}

func builtinMax(in *Interp, c *Call) Value {
	if c.N() < 1 {
		throw(c.Site, "this function expects at least one argument")
	}

	x := expectKindAt(c.At(0), c.Args[0], KInt, KFloat, KBool)

	for i := 1; i < c.N(); i++ {
		y := expectKindAt(c.At(i), c.Args[i], KInt, KFloat, KBool)

		if x.Kind() == KInt && y.Kind() == KInt {
			if intOf(y) > intOf(x) {
				x = y
			}
		} else if asFloat(y) > asFloat(x) {
			x = y
		}
	}

	return x
}

func builtinMin(in *Interp, c *Call) Value {
	if c.N() < 1 {
		throw(c.Site, "this function expects at least one argument")
	}

	x := expectKindAt(c.At(0), c.Args[0], KInt, KFloat, KBool)

	for i := 1; i < c.N(); i++ {
		y := expectKindAt(c.At(i), c.Args[i], KInt, KFloat, KBool)

		if x.Kind() == KInt && y.Kind() == KInt {
			if intOf(y) < intOf(x) {
				x = y
			}
		} else if asFloat(y) < asFloat(x) {
			x = y
		}
	}

	return x
}
