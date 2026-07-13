package math

import "mca/internal/interp"

// numDoc is the shared doc shape for the plain one-argument numeric functions
// (sin, cos, sqrt, ...): int|float|bool in, int|float out.
func numDoc(name, description string) interp.Doc {
	return interp.Doc{
		Params:      []interp.Param{{Name: "x", Type: "int|float|bool"}},
		Returns:     "int|float",
		Description: description,
		Examples:    []string{"math." + name + "(0)"},
	}
}

var docs = map[string]interp.Doc{
	"PI": {
		Returns:     "float",
		Description: "The constant pi (3.14159...).",
		Examples:    []string{`math.PI()  ->  3.141593`},
	},
	"E": {
		Returns:     "float",
		Description: "The constant e (2.71828...).",
		Examples:    []string{`math.E()  ->  2.718282`},
	},
	"abs": {
		Params:      []interp.Param{{Name: "x", Type: "int|float|bool"}},
		Returns:     "int|float",
		Description: "Absolute value of x. Returns an int if x is an int or bool, a float if x is a float.",
		Examples: []string{
			`math.abs(-15.5)  ->  15.5`,
			`math.abs(-1)  ->  1`,
		},
	},
	"sin":  numDoc("sin", "Sine of x, in radians."),
	"cos":  numDoc("cos", "Cosine of x, in radians."),
	"asin": numDoc("asin", "Arcsine of x, result in radians."),
	"acos": numDoc("acos", "Arccosine of x, result in radians."),
	"tan":  numDoc("tan", "Tangent of x, in radians."),
	"rad": {
		Params:      []interp.Param{{Name: "degrees", Type: "int|float|bool"}},
		Returns:     "int|float",
		Description: "Converts degrees to radians.",
		Examples:    []string{`math.rad(180)  ->  3.141593`},
	},
	"deg": {
		Params:      []interp.Param{{Name: "radians", Type: "int|float|bool"}},
		Returns:     "int|float",
		Description: "Converts radians to degrees.",
		Examples:    []string{`math.deg(math.PI())  ->  180`},
	},
	"sqrt":  numDoc("sqrt", "Square root of x."),
	"log":   numDoc("log", "Natural logarithm (base e) of x."),
	"log10": numDoc("log10", "Base-10 logarithm of x."),
	"exp":   numDoc("exp", "e raised to the power of x."),
	"floor": numDoc("floor", "x rounded down to the nearest whole number."),
	"ceil":  numDoc("ceil", "x rounded up to the nearest whole number."),
	"round": numDoc("round", "x rounded to the nearest whole number (half away from zero)."),
}
