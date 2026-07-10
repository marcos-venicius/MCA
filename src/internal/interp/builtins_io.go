package interp

import (
	"os"

	"mca/internal/ast"
)

func builtinPrint(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	last := UnitV()

	for _, arg := range args {
		last = in.Eval(arg).Value
		printValue(in, last, false)
	}

	return last
}

func builtinPrintln(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	last := UnitV()

	for i, arg := range args {
		if i > 0 {
			writeOut(in, " ")
		}

		last = in.Eval(arg).Value
		printValue(in, last, false)
	}

	writeOut(in, "\n")

	return last
}

func writeOut(in *Interp, s string) {
	_, _ = in.Out.Write([]byte(s))
}

func builtinExit(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	code := intOf(expectKind(args[0], in.Eval(args[0]).Value, KInt))
	os.Exit(int(code))
	panic("unreachable")
}

// TODO: make file path relative when starting with '.'
func builtinReadEntireFile(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	path := stringOf(expectKind(args[0], in.Eval(args[0]).Value, KString))

	content, err := os.ReadFile(path)
	if err != nil {
		throw(caller.Pos(), "could not read file '%s': %s", path, err)
	}

	return StringV(string(content))
}
