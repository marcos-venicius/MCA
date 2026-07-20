package interp

// Module is a native package: a group of builtins that a program only gets
// hold of by asking for it, `const crypt = import('crypt')`. Nothing about a
// module is special at the language level -- import() hands back an ordinary
// map of name -> builtin, and `crypt.md5(s)` is the same dot-access on a map
// that a file module written in MCA already gets. That is the whole reason
// packages cost so little: the value shape already existed.
//
// A module lives in its own Go package under internal/packages/<name>, and
// registers itself from init(); internal/packages then blank-imports each of
// them, and cmd/mca blank-imports that. The dependency therefore only ever
// points *at* interp, never out of it, which is what keeps a package free to
// use interp's builtin-authoring API without an import cycle.
//
// Docs is keyed by the bare function name ("md5"), not the qualified one --
// help() qualifies it as "crypt.md5" when it looks the entry up. ConstantDocs
// is the same idea for Constants: keyed by the bare constant name, holding a
// one-line explanation help() shows next to the value. Both are optional --
// a missing entry just means help() prints the member without a gloss.
type Module struct {
	Name         string
	Fns          map[string]*Native
	Docs         map[string]Doc
	Constants    map[string]Value
	ConstantDocs map[string]string
}

// nativeModules is every registered package, by the name import() takes.
// Populated from package init()s, so it is fully built before main runs and
// is never written to afterwards.
var nativeModules = map[string]*Module{}

// RegisterModule adds m to the set of packages import() can resolve. It is
// meant to be called from a package's init(); a duplicate name is a
// programming error in the interpreter itself, not something a program can
// provoke, so it panics rather than raising an MCA runtime error.
func RegisterModule(m *Module) {
	if _, dup := nativeModules[m.Name]; dup {
		panic("interp: native module '" + m.Name + "' registered twice")
	}
	nativeModules[m.Name] = m
}

// NewNative builds one builtin for a module's Fns table. Name is the
// qualified name ("crypt.md5"), since that is what error messages and
// help(crypt.md5) show. Arity is the exact argument count the shared call
// path enforces before the implementation runs; -1 means variadic.
func NewNative(name string, arity int, fn BuiltinFn) *Native {
	return native(name, arity, fn)
}

// moduleValue builds the map import() returns for m: its functions and its
// constants, under their bare names.
//
// The map is frozen before it is returned, because a package's members --
// functions and constants alike -- are constants: `io.O_RDONLY = 1` or
// `crypt.md5 = 1` is a runtime error, not a silent clobber. Freezing also
// means the map can safely be handed out fresh per call without any importer
// reaching another's copy. The *Native values inside are shared and immutable,
// so this is one small allocation per import, not a deep copy of the package.
func moduleValue(m *Module) *Map {
	mp := NewMap()
	for name, n := range m.Fns {
		mp.Set(MapKey{Kind: KString, S: name}, FnValV(&FnValue{Native: n}))
	}
	for name, v := range m.Constants {
		mp.Set(MapKey{Kind: KString, S: name}, v)
	}
	mp.Freeze()
	return mp
}

// splitQualified splits "crypt.md5" into its module and function halves.
// Reports ok only when both halves are non-empty, so "crypt", ".md5" and
// "crypt." are all left for the caller to reject.
func splitQualified(name string) (mod, fn string, ok bool) {
	for i := 0; i < len(name); i++ {
		if name[i] == '.' {
			mod, fn = name[:i], name[i+1:]
			return mod, fn, mod != "" && fn != ""
		}
	}
	return "", "", false
}
