package interp

// TODO: since we have GO's GC on our behalf, do we need to have a map clear?
//       What if we just assign an empty object to the map?

func builtinMapDel(in *Interp, c *Call) Value {
	m := expectKindAt(c.At(0), c.Args[0], KMap).(*Map)
	key := expectKindAt(c.At(1), c.Args[1], KInt, KString)

	mk, _ := mapKeyFromValue(key)

	return BoolV(m.Del(mk))
}

func builtinMapClear(in *Interp, c *Call) Value {
	m := expectKindAt(c.At(0), c.Args[0], KMap).(*Map)
	m.Clear()
	return UnitV()
}

func builtinMapKeys(in *Interp, c *Call) Value {
	m := expectKindAt(c.At(0), c.Args[0], KMap).(*Map)

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

func builtinMapValues(in *Interp, c *Call) Value {
	m := expectKindAt(c.At(0), c.Args[0], KMap).(*Map)

	values := make([]Value, 0, m.Len())

	for _, v := range m.values {
		values = append(values, v)
	}

	out := Array{
		Items: values,
	}

	return ArrayV(&out)
}
