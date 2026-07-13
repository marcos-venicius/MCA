package interp

import (
	"os"
	"path/filepath"
	"strings"

	"mca/internal/lexer"
	"mca/internal/parser"
)

func builtinArgc(in *Interp, c *Call) Value {
	return IntV(int64(len(in.Args)))
}

func builtinArgv(in *Interp, c *Call) Value {
	idx := intOf(expectKindAt(c.At(0), c.Args[0], KInt))

	if idx < 0 || idx >= int64(len(in.Args)) {
		throw(c.Site, "index %d is out of range. You have %d arguments.", idx, len(in.Args))
	}

	return StringV(in.Args[idx])
}

// isModulePath reports whether the argument to import() names a *file* rather
// than a native package. The split is purely syntactic, and is the one rule
// worth being rigid about, because programs are written against it:
//
//   - starts with '.'   -- './lexer.mca', relative to the importing file
//   - absolute          -- '/opt/mca/lexer.mca'
//   - ends in '.mca'    -- 'lib/lexer.mca', relative to the working directory
//
// Anything else ('crypt') is a package name, and is never looked for on disk.
// A bare name therefore cannot be silently shadowed by a file that happens to
// sit next to the program, and adding a package can never change what an
// existing file import resolves to.
func isModulePath(s string) bool {
	return strings.HasPrefix(s, ".") || filepath.IsAbs(s) || strings.HasSuffix(s, ".mca")
}

// builtinImport implements import(), which resolves one of two things.
//
// A bare name is a native package (internal/packages/...): import() hands back
// a fresh map of its builtins, so `const crypt = import('crypt')` then
// `crypt.md5(s)` is ordinary dot-access on an ordinary map. Nothing is loaded
// from disk and nothing is evaluated.
//
// A path (see isModulePath) is an MCA file, whose value is whatever its last
// statement evaluated to -- by convention a map of the names it means to
// export. A `.`-prefixed path resolves relative to the *importing file's own
// directory* (via c.Site.Filename), not the working directory. Every call
// re-lexes/re-parses/re-runs the file from scratch in a brand new Interp with
// no shared environment and no caching.
// TODO: I didn't tested this code cross-platform. I have no idea of how this is gonna behave on Windows for example.
// TODO: We're not caching the import evaluation. For now, Everytime you import the same module it's gaonna be reevaluated.
func builtinImport(in *Interp, c *Call) Value {
	pathArg := c.StringArg(0)

	if pathArg == "" {
		throw(c.At(0), "invalid module path '%s'", pathArg)
	}

	if !isModulePath(pathArg) {
		m, ok := nativeModules[pathArg]
		if !ok {
			throw(c.At(0), "there is no package named '%s' -- run help() to list the packages you can import, or import a file as './%s.mca'", pathArg, pathArg)
		}
		return moduleValue(m)
	}

	resolved := pathArg

	if strings.HasPrefix(pathArg, ".") {
		callerFile := c.Site.Filename
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
		throw(c.Site, "could not open module '%s' due to: '%s'", resolved, err)
	}

	l := lexer.New(resolved, string(content))
	tokens := l.Tokenize()
	if len(l.Errors) > 0 {
		throw(c.Site, "could not read module %s", resolved)
	}

	prog := parser.Parse(resolved, tokens)
	if len(prog.Errors) > 0 {
		throw(c.Site, "could not parse module %s", resolved)
	}

	moduleInterp := New()

	// runStatements (not Run) deliberately has no recover boundary: a
	// runtime error inside the module must panic straight through to the
	// program's single outermost recover, not be swallowed here.
	return moduleInterp.runStatements(prog.Stmts)
}
