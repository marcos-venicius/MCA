package interp

import (
	"os"
	"path/filepath"
	"strings"

	"mca/internal/ast"
	"mca/internal/lexer"
	"mca/internal/parser"
)

func builtinArgc(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	return IntV(int64(len(in.Args)))
}

func builtinArgv(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	idx := intOf(expectKind(args[0], in.Eval(args[0]).Value, KInt))

	if idx < 0 || idx >= int64(len(in.Args)) {
		throw(caller.Pos(), "index %d is out of range. You have %d arguments.", idx, len(in.Args))
	}

	return StringV(in.Args[idx])
}

// builtinImport implements import(): a `.`-prefixed path resolves relative
// to the *importing file's own directory* (via caller.Pos().Filename),
// otherwise the raw path is used as-is (room for a future stdlib search
// path). Every call re-lexes/re-parses/re-runs the file from scratch in a
// brand new Interp with no shared environment and no caching.
// TODO: I didn't tested this code cross-platform. I have no idea of how this is gonna behave on Windows for example.
func builtinImport(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	pathArg := stringOf(expectKind(args[0], in.Eval(args[0]).Value, KString))

	if pathArg == "" {
		throw(args[0].Pos(), "invalid module path '%s'", pathArg)
	}

	resolved := pathArg

	if strings.HasPrefix(pathArg, ".") {
		callerFile := caller.Pos().Filename
		if callerFile != "" {
			combined := filepath.Join(filepath.Dir(callerFile), pathArg)
			if abs, err := filepath.Abs(combined); err == nil {
				resolved = abs
			} else {
				resolved = combined
			}
		}
	}

	content, err := os.ReadFile(resolved)
	if err != nil {
		throw(caller.Pos(), "could not open module '%s' due to: '%s'", resolved, err)
	}

	l := lexer.New(resolved, string(content))
	tokens := l.Tokenize()
	if len(l.Errors) > 0 {
		throw(caller.Pos(), "could not read module %s", resolved)
	}

	prog := parser.Parse(resolved, tokens)
	if len(prog.Errors) > 0 {
		throw(caller.Pos(), "could not parse module %s", resolved)
	}

	moduleInterp := New()

	// runStatements (not Run) deliberately has no recover boundary: a
	// runtime error inside the module must panic straight through to the
	// program's single outermost recover, not be swallowed here.
	return moduleInterp.runStatements(prog.Stmts)
}
