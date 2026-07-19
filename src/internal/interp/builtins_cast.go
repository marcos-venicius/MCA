package interp

import (
	"strconv"

	"mca/internal/ast"
)

func isTypeBuiltin(kind Kind) BuiltinFn {
	return func(in *Interp, c *Call) Value {
		v := c.Args[0]
		return BoolV(v.Kind() == kind)
	}
}

func builtinType(in *Interp, c *Call) Value {
	return StringV(c.Args[0].Kind().String())
}

func builtinAsInt(in *Interp, c *Call) Value {
	v := c.Args[0]

	switch v.Kind() {
	case KInt:
		return v
	case KFloat:
		return IntV(int64(floatOf(v)))
	case KBool:
		return IntV(boolToInt(boolOf(v)))
	case KString:
		s := stringOf(v)
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			throwNumError(c.At(0), err, s, KInt.String())
		}
		return IntV(n)
	}

	throw(c.At(0), "cannot cast '%s' to int", v.Kind())
	panic("unreachable")
}

func builtinAsFloat(in *Interp, c *Call) Value {
	v := c.Args[0]

	switch v.Kind() {
	case KInt:
		return FloatV(float64(intOf(v)))
	case KFloat:
		return v
	case KBool:
		if boolOf(v) {
			return FloatV(1)
		}
		return FloatV(0)
	case KString:
		s := stringOf(v)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			throwNumError(c.At(0), err, s, "float")
		}
		return FloatV(f)
	}

	throw(c.At(0), "cannot cast '%s' to float", v.Kind())
	panic("unreachable")
}

func builtinAsBool(in *Interp, c *Call) Value {
	v := c.Args[0]

	if result, ok := Truthy(v); ok {
		return BoolV(result)
	}

	throw(c.At(0), "cannot cast '%s' to bool", v.Kind())
	panic("unreachable")
}

func builtinAsString(in *Interp, c *Call) Value {
	v := c.Args[0]

	// TODO: implement for maps and arrays too (something like javascript does)
	switch v.Kind() {
	case KInt:
		return StringV(strconv.FormatInt(intOf(v), 10))
	case KFloat:
		return StringV(strconv.FormatFloat(floatOf(v), 'f', 6, 64))
	case KBool:
		if boolOf(v) {
			return StringV("true")
		}
		return StringV("false")
	case KString:
		return v
	}

	throw(c.At(0), "cannot cast '%s' to string", v.Kind())
	panic("unreachable")
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// throwNumError reports as_int/as_float's error-message split between
// out-of-range and not-a-valid-number.
func throwNumError(pos ast.Pos, err error, s, kind string) {
	if numErr, ok := err.(*strconv.NumError); ok && numErr.Err == strconv.ErrRange {
		throw(pos, "the number is too large or too small to fit in a %s type", kind)
	}
	throw(pos, "'%s' is not a valid %s literal", s, kind)
}
