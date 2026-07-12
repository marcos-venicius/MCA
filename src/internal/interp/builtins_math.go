package interp

import (
	"math"

	"mca/internal/ast"
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
	return func(in *Interp, caller ast.Expr, args []ast.Expr) Value {
		arg := expectKind(args[0], in.Eval(args[0]).Value, KInt, KFloat, KBool)
		result := f(asFloat(arg))

		if math.Mod(result, 1.0) != 0.0 {
			return FloatV(result)
		}
		return IntV(int64(result))
	}
}

// TODO: later I need to think about constants
func builtinPI(in *Interp, caller ast.Expr, args []ast.Expr) Value { return FloatV(math.Pi) }
func builtinE(in *Interp, caller ast.Expr, args []ast.Expr) Value  { return FloatV(math.E) }

func builtinSum(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	arr := expectKind(args[0], in.Eval(args[0]).Value, KArray).(*Array)

	hasFloat := false

	var r float64 = 0

	for _, v := range arr.Items {
		if v.Kind() != KInt && v.Kind() != KFloat {
			throw(args[0].Pos(), "expected int | float but got %s", v.Kind())
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

func builtinAbs(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	arg := expectKind(args[0], in.Eval(args[0]).Value, KInt, KFloat, KBool)

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

func builtinMax(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	if len(args) < 1 {
		throw(caller.Pos(), "this function expects at least one argument")
	}

	x := expectKind(args[0], in.Eval(args[0]).Value, KInt, KFloat, KBool)

	for i := 1; i < len(args); i++ {
		y := expectKind(args[i], in.Eval(args[i]).Value, KInt, KFloat, KBool)

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

func builtinMin(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	if len(args) < 1 {
		throw(caller.Pos(), "this function expects at least one argument")
	}

	x := expectKind(args[0], in.Eval(args[0]).Value, KInt, KFloat, KBool)

	for i := 1; i < len(args); i++ {
		y := expectKind(args[i], in.Eval(args[i]).Value, KInt, KFloat, KBool)

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
