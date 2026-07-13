package interp

import (
	"math"
)

func calcRad(degrees float64) float64 { return degrees * (math.Pi / 180.0) }
func calcDeg(radians float64) float64 { return radians * (180.0 / math.Pi) }

// mathBuiltin adapts a plain float64->float64 math function into a
// BuiltinFn: accepts int/float/bool, computes in float64, and collapses back
// to int if the result has zero fractional part (so e.g. sin(0) is int 0,
// not float 0.0).
// TODO: Is this behavior fine? I need to investigate better how languages like python
// handles this problem.
func mathBuiltin(f func(float64) float64) BuiltinFn {
	return func(in *Interp, c *Call) Value {
		arg := expectKindAt(c.At(0), c.Args[0], KInt, KFloat, KBool)
		result := f(asFloat(arg))

		if math.Mod(result, 1.0) != 0.0 {
			return FloatV(result)
		}
		return IntV(int64(result))
	}
}

// TODO: later I need to think about constants
func builtinPI(in *Interp, c *Call) Value { return FloatV(math.Pi) }
func builtinE(in *Interp, c *Call) Value  { return FloatV(math.E) }

func builtinSum(in *Interp, c *Call) Value {
	arr := expectKindAt(c.At(0), c.Args[0], KArray).(*Array)

	hasFloat := false

	var r float64 = 0

	for _, v := range arr.Items {
		if v.Kind() != KInt && v.Kind() != KFloat {
			throw(c.At(0), "expected int | float but got %s", v.Kind())
		}

		if v.Kind() == KFloat {
			hasFloat = true

			r += float64(v.(FloatValue))
		} else {
			r += float64(v.(IntValue))
		}
	}

	if hasFloat {
		return FloatV(r)
	}

	return IntV(int64(r))
}

func builtinAbs(in *Interp, c *Call) Value {
	arg := expectKindAt(c.At(0), c.Args[0], KInt, KFloat, KBool)

	if arg.Kind() == KInt || arg.Kind() == KBool {
		return IntV(absInt64(asIntLike(arg)))
	}
	return FloatV(math.Abs(floatOf(arg)))
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
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
