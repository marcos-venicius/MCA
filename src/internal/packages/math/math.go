// Package math is MCA's `math` package: the numeric functions that used to be
// always-bound builtins, reached with `const math = import('math')`. Only
// sum, min and max stayed behind as builtins -- they are everyday list
// operations more than they are mathematics.
//
// The numeric behavior is unchanged from the builtin era: every function
// accepts int, float or bool (bools count as 0/1), computes in float64, and
// collapses the result back to an int when it has no fractional part (so
// math.sin(0) is int 0, not float 0.0).
package math

import (
	"math"

	"mca/internal/interp"
)

func init() {
	interp.RegisterModule(&interp.Module{
		Name: "math",
		Fns: map[string]*interp.Native{
			"PI":    interp.NewNative("math.PI", 0, pi),
			"E":     interp.NewNative("math.E", 0, e),
			"abs":   interp.NewNative("math.abs", 1, abs),
			"sin":   interp.NewNative("math.sin", 1, fn1(math.Sin)),
			"cos":   interp.NewNative("math.cos", 1, fn1(math.Cos)),
			"asin":  interp.NewNative("math.asin", 1, fn1(math.Asin)),
			"acos":  interp.NewNative("math.acos", 1, fn1(math.Acos)),
			"tan":   interp.NewNative("math.tan", 1, fn1(math.Tan)),
			"rad":   interp.NewNative("math.rad", 1, fn1(calcRad)),
			"deg":   interp.NewNative("math.deg", 1, fn1(calcDeg)),
			"sqrt":  interp.NewNative("math.sqrt", 1, fn1(math.Sqrt)),
			"log":   interp.NewNative("math.log", 1, fn1(math.Log)),
			"log10": interp.NewNative("math.log10", 1, fn1(math.Log10)),
			"exp":   interp.NewNative("math.exp", 1, fn1(math.Exp)),
			"floor": interp.NewNative("math.floor", 1, fn1(math.Floor)),
			"ceil":  interp.NewNative("math.ceil", 1, fn1(math.Ceil)),
			"round": interp.NewNative("math.round", 1, fn1(math.Round)),
		},
		Docs: docs,
	})
}

func calcRad(degrees float64) float64 { return degrees * (math.Pi / 180.0) }
func calcDeg(radians float64) float64 { return radians * (180.0 / math.Pi) }

// numArg is argument i widened to float64, accepting int, float, or bool
// (bools count as 0/1) -- the same numeric coercion these functions applied
// when they were builtins.
func numArg(c *interp.Call, i int) float64 {
	v := c.Arg(i, interp.KInt, interp.KFloat, interp.KBool)

	switch vv := v.(type) {
	case interp.FloatValue:
		return float64(vv)
	case interp.BoolValue:
		if vv {
			return 1
		}
		return 0
	default: // interp.IntValue
		return float64(vv.(interp.IntValue))
	}
}

// fn1 adapts a plain float64->float64 function into a package function:
// one numeric argument, computed in float64, collapsed back to int if the
// result has zero fractional part.
func fn1(f func(float64) float64) interp.BuiltinFn {
	return func(in *interp.Interp, c *interp.Call) interp.Value {
		result := f(numArg(c, 0))

		if math.Mod(result, 1.0) != 0.0 {
			return interp.FloatV(result)
		}
		return interp.IntV(int64(result))
	}
}

// TODO: later I need to think about constants
func pi(in *interp.Interp, c *interp.Call) interp.Value { return interp.FloatV(math.Pi) }
func e(in *interp.Interp, c *interp.Call) interp.Value  { return interp.FloatV(math.E) }

func abs(in *interp.Interp, c *interp.Call) interp.Value {
	v := c.Arg(0, interp.KInt, interp.KFloat, interp.KBool)

	if fv, ok := v.(interp.FloatValue); ok {
		return interp.FloatV(math.Abs(float64(fv)))
	}

	// int or bool: stay integral
	n := int64(numArg(c, 0))
	if n < 0 {
		n = -n
	}
	return interp.IntV(n)
}
