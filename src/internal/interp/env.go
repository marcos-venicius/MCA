package interp

// binding is one variable slot. Bindings are stored by pointer so that
// Assign can rewrite a slot found several frames up the parent chain without
// having to re-index the map it lives in.
type binding struct {
	value   Value
	isConst bool
}

// Env is a lexically-scoped variable frame. There is
// no ref-counting here the way C needed, Go's GC keeps an Env reachable for
// exactly as long as something (a parent chain, or a closure's captured Env)
// still points to it.
type Env struct {
	vars   map[string]*binding
	parent *Env
}

func NewEnv(parent *Env) *Env {
	return &Env{vars: make(map[string]*binding), parent: parent}
}

// Get climbs the parent chain looking for an existing binding.
func (e *Env) Get(name string) (Value, bool) {
	if b, ok := e.lookup(name); ok {
		return b.value, true
	}
	return nil, false
}

// lookup climbs the parent chain and returns the binding itself, so callers
// can inspect its constness or overwrite it in place.
func (e *Env) lookup(name string) (*binding, bool) {
	for env := e; env != nil; env = env.parent {
		if b, ok := env.vars[name]; ok {
			return b, true
		}
	}
	return nil, false
}

// HasLocal reports whether name is bound in *this exact scope*, ignoring
// enclosing ones. Used to reject redeclaring a constant in the scope that
// already owns it, while still allowing an inner scope to shadow it.
func (e *Env) HasLocal(name string) bool {
	_, ok := e.vars[name]
	return ok
}

// Define always writes into this exact scope, never climbing
// used for function parameters and
// loop-bound variables (k/v, range index), which always shadow outer scopes.
func (e *Env) Define(name string, v Value) {
	e.vars[name] = &binding{value: v}
}

// DefineConst is Define for a `const name = ...` declaration: same
// always-this-scope placement, but the resulting binding rejects every later
// Assign. Constness is a property of the *binding*, not of the value, so
// `const a = [1]` still permits `a[0] = 2` -- the name is frozen, the array
// it points at is not.
func (e *Env) DefineConst(name string, v Value) {
	e.vars[name] = &binding{value: v, isConst: true}
}

// IsConst reports whether name resolves to a constant binding.
func (e *Env) IsConst(name string) bool {
	b, ok := e.lookup(name)
	return ok && b.isConst
}

// Assign climbs the parent chain to find an existing binding to mutate in
// place; if none exists anywhere, it implicitly creates the variable in
// *this* scope i.e. Python-like
// implicit-declare-on-first-assign semantics.
//
// It returns false, writing nothing, when the binding it lands on is a
// constant -- the caller turns that into a runtime error, since it has the
// source position needed to report one.
func (e *Env) Assign(name string, v Value) bool {
	if b, ok := e.lookup(name); ok {
		if b.isConst {
			return false
		}
		b.value = v
		return true
	}

	e.vars[name] = &binding{value: v}
	return true
}
