package interp

import (
	"os"
)

func builtinPrint(in *Interp, c *Call) Value {
	last := UnitV()

	for _, arg := range c.Args {
		last = arg
		printValue(in, last, false)
	}

	return last
}

func builtinPrintln(in *Interp, c *Call) Value {
	last := UnitV()

	for i, arg := range c.Args {
		if i > 0 {
			writeOut(in, " ")
		}

		last = arg
		printValue(in, last, false)
	}

	writeOut(in, "\n")

	return last
}

func writeOut(in *Interp, s string) {
	_, _ = in.Out.Write([]byte(s))
}

func builtinExit(in *Interp, c *Call) Value {
	code := intOf(expectKindAt(c.At(0), c.Args[0], KInt))
	os.Exit(int(code))
	panic("unreachable")
}

// TODO: make file path relative when starting with '.'
func builtinReadEntireFile(in *Interp, c *Call) Value {
	path := stringOf(expectKindAt(c.At(0), c.Args[0], KString))

	content, err := os.ReadFile(path)
	if err != nil {
		throw(c.Site, "could not read file '%s': %s", path, err)
	}

	return StringV(string(content))
}
