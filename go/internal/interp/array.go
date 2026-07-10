package interp

// Array is an MCA array: a thin wrapper around a Go slice, we have have
// some additional info in the future
type Array struct {
	Items []Value
}

func (*Array) Kind() Kind { return KArray }
