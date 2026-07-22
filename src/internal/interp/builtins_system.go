package interp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mca/internal/lexer"
	"mca/internal/parser"
)

func builtinLen(in *Interp, c *Call) Value {
	v := expectKindAt(c.At(0), c.Args[0], KString, KMap, KArray)

	switch v.Kind() {
	case KString:
		return IntV(int64(v.num))
	case KMap:
		return IntV(int64(mapOf(v).Len()))
	default: // KArray
		return IntV(int64(len(arrayOf(v).Items)))
	}
}

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
//   - ends in '.mca'    -- 'lib/lexer.mca', looked up on the search path
//
// Anything else ('crypt') is a package name, and is never looked for on disk.
// A bare name therefore cannot be silently shadowed by a file that happens to
// sit next to the program, and adding a package can never change what an
// existing file import resolves to.
func isModulePath(s string) bool {
	return strings.HasPrefix(s, ".") || filepath.IsAbs(s) || strings.HasSuffix(s, ".mca")
}

// searchPathEnv is the environment variable that holds import()'s search path:
// a list of directories, in the platform's PATH format (':'-separated on Unix),
// searched in order for a "complete name" import like 'json.mca'.
const searchPathEnv = "MCA_SEARCH_PATHS"

// moduleSearchPaths returns the directories listed in MCA_SEARCH_PATHS, in the
// order they were given, with empty entries dropped. An unset or empty variable
// yields no paths, in which case a complete-name import falls back to its old
// working-directory-relative behavior.
func moduleSearchPaths() []string {
	raw := os.Getenv(searchPathEnv)
	if raw == "" {
		return nil
	}

	var dirs []string
	for dir := range strings.SplitSeq(raw, string(os.PathListSeparator)) {
		if dir != "" {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// ResolvePath resolves a user-supplied file path the way import() resolves a
// module path: a path starting with '.' is relative to the *calling file's own
// directory* (c.Site.Filename), not the process working directory; any other
// path is returned unchanged. Exported so a native package that touches the
// filesystem (io.read_entire_file) resolves paths identically to import().
func (c *Call) ResolvePath(path string) string {
	if !strings.HasPrefix(path, ".") {
		return path
	}

	callerFile := c.Site.Filename
	if callerFile == "" {
		return path
	}

	combined := filepath.Join(filepath.Dir(callerFile), path)
	if abs, err := filepath.Abs(combined); err == nil {
		return abs
	}
	return combined
}

// resolveImportPath resolves the argument to import() to a file on disk. It
// extends ResolvePath with one rule that only import() gets -- deliberately not
// the shared filesystem helpers, so io.read_entire_file and friends keep
// resolving paths exactly as before:
//
// A "complete name" -- one that reaches here ending in '.mca' but neither
// '.'-prefixed nor absolute, e.g. 'json.mca' or 'lib/json.mca' -- is looked up
// against the MCA_SEARCH_PATHS directories, in order, and the first one that
// holds it wins. If no search path holds it (or none are configured), the name
// falls back to the ordinary ResolvePath behavior: working-directory-relative,
// exactly as it resolved before the search path existed.
//
// '.'-prefixed and absolute paths never touch the search path; they defer
// straight to ResolvePath.
func (c *Call) resolveImportPath(path string) string {
	if strings.HasPrefix(path, ".") || filepath.IsAbs(path) {
		return c.ResolvePath(path)
	}

	for _, dir := range moduleSearchPaths() {
		candidate := filepath.Join(dir, path)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	return c.ResolvePath(path)
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
// directory* (via c.Site.Filename); an absolute path is used as-is; and a
// complete name like 'json.mca' is looked up on MCA_SEARCH_PATHS (see
// resolveImportPath). Every call re-lexes/re-parses/re-runs the file from
// scratch in a brand new Interp with no shared environment and no caching.
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
		return MapV(moduleValue(m))
	}

	resolved := c.resolveImportPath(pathArg)

	content, err := os.ReadFile(resolved)
	if err != nil {
		throw(c.Site, "could not open module '%s' due to: '%s'", resolved, err)
	}

	l := lexer.New(resolved, string(content))
	tokens := l.Tokenize()
	if len(l.Errors) > 0 {
		for _, e := range l.Errors {
			fmt.Fprintln(os.Stderr, e.Error())
		}

		throw(c.Site, "could not read module %s", resolved)
	}

	prog := parser.Parse(resolved, tokens)
	if len(prog.Errors) > 0 {
		for _, e := range prog.Errors {
			fmt.Fprintln(os.Stderr, e.Error())
		}

		fmt.Fprintf(os.Stderr, "parsing failed with \033[1;31m%d\033[0m errors\n", len(prog.Errors))

		throw(c.Site, "could not parse module %s", resolved)
	}

	moduleInterp := New()

	// runStatements (not Run) deliberately has no recover boundary: a
	// runtime error inside the module must panic straight through to the
	// program's single outermost recover, not be swallowed here.
	return moduleInterp.runStatements(prog.Stmts)
}
