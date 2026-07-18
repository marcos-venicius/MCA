package interp

// binding is one variable slot: its value and whether it's const.
type binding struct {
	value   Value
	isConst bool
}

// Env is a lexical scope. The resolver hands every Ident a (depth, slot) pair,
// so the common path -- bySlot -- is a fixed number of parent hops plus an
// array index, no hashing. names is kept only for the fallback the resolver
// can't cover statically (Depth == -1): builtins, forward references, and
// anything else that has to be found by name at runtime.
//
// Go's GC keeps an Env alive for exactly as long as something -- a parent
// chain or a closure that captured it -- still points at it.
type Env struct {
	slots  []*binding
	names  map[string]int
	parent *Env
}

func NewEnv(parent *Env) *Env {
	return &Env{names: map[string]int{}, parent: parent}
}

// NewBuiltinEnv is the root scope the builtins live in; the global scope is its
// child. Builtins are only ever reached by name (their uses resolve to
// Depth == -1), so this scope exists purely for the byName fallback to bottom
// out in.
func NewBuiltinEnv() *Env {
	return NewEnv(nil)
}

// define binds name in this exact scope at the resolver-assigned slot,
// shadowing any outer binding of the same name.
func (e *Env) define(slot int, name string, v Value, isConst bool) {
	for len(e.slots) <= slot {
		e.slots = append(e.slots, nil)
	}
	e.slots[slot] = &binding{value: v, isConst: isConst}
	e.names[name] = slot
}

// bySlot follows depth parent links and returns the binding at slot, or nil if
// nothing lives there yet (a slot read before its declaring assignment ran).
func (e *Env) bySlot(depth, slot int) *binding {
	env := e
	for i := 0; i < depth && env != nil; i++ {
		env = env.parent
	}
	if env == nil || slot < 0 || slot >= len(env.slots) {
		return nil
	}
	return env.slots[slot]
}

// byName climbs the parent chain looking for name. Used for the Depth == -1
// fallback and nowhere on the hot path.
func (e *Env) byName(name string) (*binding, bool) {
	for env := e; env != nil; env = env.parent {
		if slot, ok := env.names[name]; ok {
			return env.slots[slot], true
		}
	}
	return nil, false
}

// hasLocal reports whether name is bound in this exact scope, so declareConst
// can reject redeclaring a constant next to itself while still letting an inner
// scope shadow it.
func (e *Env) hasLocal(name string) bool {
	_, ok := e.names[name]
	return ok
}

// assign writes v to the binding the resolver picked out. An existing binding
// is mutated in place (rejected, with a false return, if it's const); a target
// that isn't live yet is created here -- that covers both implicit
// declare-on-first-assign and shadowing a builtin, since the resolver hands
// those a fresh local slot in the current scope.
func (e *Env) assign(depth, slot int, name string, v Value) bool {
	if b := e.bySlot(depth, slot); b != nil {
		if b.isConst {
			return false
		}
		b.value = v
		return true
	}

	e.define(slot, name, v, false)
	return true
}
