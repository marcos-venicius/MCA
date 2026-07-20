package interp

// Removing a key is delete(m, key) -- the same builtin that removes array
// elements; there is no separate map_del/map_clear anymore (clearing is just
// rebinding to {}).

func builtinMapKeys(in *Interp, c *Call) Value {
	m := mapOf(expectKindAt(c.At(0), c.Args[0], KMap))

	values := make([]Value, 0, m.Len())

	for k := range m.values {
		value := mapValueFromKey(k)

		values = append(values, value)
	}

	out := Array{
		Items: values,
	}

	return ArrayV(&out)
}

// builtinFreeze marks a map immutable and returns it. The freeze is in place --
// the argument and the returned value are the same map -- so a program can
// freeze a value it already holds or freeze one inline. This is the same
// mechanism native packages use for their (constant) members, exposed so a
// module written in MCA can make its own map read-only.
func builtinFreeze(in *Interp, c *Call) Value {
	m := mapOf(expectKindAt(c.At(0), c.Args[0], KMap))
	m.Freeze()
	return c.Args[0]
}

func builtinMapValues(in *Interp, c *Call) Value {
	m := mapOf(expectKindAt(c.At(0), c.Args[0], KMap))

	values := make([]Value, 0, m.Len())

	for _, v := range m.values {
		values = append(values, v)
	}

	out := Array{
		Items: values,
	}

	return ArrayV(&out)
}
