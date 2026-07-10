package interp

// Env is a lexically-scoped variable frame. There is
// no ref-counting here the way C needed, Go's GC keeps an Env reachable for
// exactly as long as something (a parent chain, or a closure's captured Env)
// still points to it.
type Env struct {
	vars   map[string]Value
	parent *Env
}

func NewEnv(parent *Env) *Env {
	return &Env{vars: make(map[string]Value), parent: parent}
}

// Get climbs the parent chain looking for an existing binding.
func (e *Env) Get(name string) (Value, bool) {
	for env := e; env != nil; env = env.parent {
		if v, ok := env.vars[name]; ok {
			return v, true
		}
	}
	return nil, false
}

// Define always writes into this exact scope, never climbing
// used for function parameters and
// loop-bound variables (k/v, range index), which always shadow outer scopes.
func (e *Env) Define(name string, v Value) {
	e.vars[name] = v
}

// Assign climbs the parent chain to find an existing binding to mutate in
// place; if none exists anywhere, it implicitly creates the variable in
// *this* scope i.e. Python-like
// implicit-declare-on-first-assign semantics.
func (e *Env) Assign(name string, v Value) {
	for env := e; env != nil; env = env.parent {
		if _, ok := env.vars[name]; ok {
			env.vars[name] = v
			return
		}
	}

	e.vars[name] = v
}
