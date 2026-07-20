package interp

import "fmt"

type Map struct {
	values map[MapKey]Value
	frozen bool
}

type MapKey struct {
	Kind Kind // KInt, KFloat or KString
	F    float64
	I    int64
	S    string
}

func (mk MapKey) String() string {
	switch mk.Kind {
	case KInt:
		return fmt.Sprintf("%d", mk.I)
	case KFloat:
		return fmt.Sprintf("%g", mk.F)
	case KString:
		return mk.S
	}
	panic("unreacheable")
}

func isValidMapKeyType(kind Kind) bool {
	switch kind {
	case KInt, KFloat, KString:
		return true
	}
	return false
}

func NewMap() *Map {
	return &Map{values: make(map[MapKey]Value)}
}

func (m *Map) Get(k MapKey) (Value, bool) {
	v, ok := m.values[k]
	return v, ok
}

func (m *Map) Set(k MapKey, v Value) {
	m.values[k] = v
}

// Freeze marks m immutable to user code: the interpreter's assignment and
// delete paths refuse to mutate a frozen map. It is one-way and guards only
// those user-facing paths -- internal construction via Set is unaffected, so a
// map is fully built first and frozen last. Used for native package maps,
// whose members (functions and constants alike) are read-only.
func (m *Map) Freeze() { m.frozen = true }

// Frozen reports whether Freeze has been called on m.
func (m *Map) Frozen() bool { return m.frozen }

func (m *Map) Del(k MapKey) bool {
	if _, exists := m.values[k]; !exists {
		return false
	}

	delete(m.values, k)

	return true
}

func (m *Map) Len() int { return len(m.values) }
