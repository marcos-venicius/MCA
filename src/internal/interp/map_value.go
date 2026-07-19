package interp

import "fmt"

type Map struct {
	values map[MapKey]Value
}

type MapKey struct {
	Kind Kind // KInt or KString
	I    int64
	S    string
}

func (mk MapKey) String() string {
	switch mk.Kind {
	case KInt:
		return fmt.Sprintf("%d", mk.I)
	case KString:
		return mk.S
	}
	panic("unreacheable")
}

func isValidMapKeyType(kind Kind) bool {
	switch kind {
	case KInt, KString:
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

func (m *Map) Del(k MapKey) bool {
	if _, exists := m.values[k]; !exists {
		return false
	}

	delete(m.values, k)

	return true
}

func (m *Map) Len() int { return len(m.values) }
