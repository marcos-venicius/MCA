package interp

import (
	"slices"
	"strings"
)

func builtinIndexesToKeys(in *Interp, c *Call) Value {
	array := expectKindAt(c.At(0), c.Args[0], KArray).(*Array)
	obj := expectKindAt(c.At(1), c.Args[1], KMap).(*Map)

	out := make(map[MapKey]Value)

	for k, v := range obj.values {
		if k.Kind != KInt {
			throw(c.At(1), "'%s' is not an integer", k.String())
		}

		idx := int(k.I)

		if idx < 0 || idx >= len(array.Items) {
			throw(c.At(1), "index %d is out of range. array has %d elements", idx, len(array.Items))
		}

		if !isValidMapKeyType(v.Kind()) {
			throw(c.At(1), "%s is not a valid map key data type", v.Kind())
		}

		mk, _ := mapKeyFromValue(v)

		out[mk] = array.Items[idx]
	}

	return MapV(&Map{values: out})
}

func builtinSort(in *Interp, c *Call) Value {
	array := expectKindAt(c.At(0), c.Args[0], KArray).(*Array)
	lambda := expectKindAt(c.At(1), c.Args[1], KFn).(*FnValue)

	if !lambda.Accepts(2) {
		throw(c.At(1), "expected a function of two arguments but got one of %d", lambda.Arity())
	}

	copy := slices.Clone(array.Items)

	slices.SortFunc(copy, func(a, b Value) int {
		result := in.callFnValue(lambda, c.At(1), fnLabel(lambda), []Value{a, b})

		if result.Kind() != KInt {
			throw(c.At(1), "the sorting function should return an integer but returned %s. try `help(sort)`", result.Kind())
		}

		return int(result.(IntValue))
	})

	return ArrayV(&Array{
		Items: copy,
	})
}

func builtinReverse(in *Interp, c *Call) Value {
	value := expectKindAt(c.At(0), c.Args[0], KArray).(*Array)

	out := make([]Value, len(value.Items))

	for i := range value.Items {
		out[i] = value.Items[len(value.Items)-i-1]
	}

	arr := Array{
		Items: out,
	}

	return ArrayV(&arr)
}

func builtinConcat(in *Interp, c *Call) Value {
	out := make([]Value, 0)

	for i, arg := range c.Args {
		value := expectKindAt(c.At(i), arg, KArray).(*Array)

		out = append(out, value.Items...)
	}

	arr := Array{
		Items: out,
	}

	return ArrayV(&arr)
}

func builtinContains(in *Interp, c *Call) Value {
	target := expectKindAt(c.At(0), c.Args[0], KString, KArray, KMap)

	switch target.Kind() {
	case KString:
		substr := string(expectKindAt(c.At(1), c.Args[1], KString).(StringValue))

		return BoolV(strings.Contains(string(target.(StringValue)), substr))
	case KArray:
		value := c.Args[1]

		items := (target.(*Array)).Items

		for _, v := range items {
			if compareTwoValues(v, value) {
				return BoolV(true)
			}
		}

		return BoolV(false)
	case KMap:
		key := expectKindAt(c.At(1), c.Args[1], KString, KInt)

		mk, _ := mapKeyFromValue(key)
		m := (target.(*Map)).values

		if _, ok := m[mk]; ok {
			return BoolV(true)
		}

		return BoolV(false)
	}

	panic("builtinContains: unreacheable")
}

func builtinFilter(in *Interp, c *Call) Value {
	arr := expectKindAt(c.At(0), c.Args[0], KArray).(*Array).Items
	fn := expectKindAt(c.At(1), c.Args[1], KFn).(*FnValue)

	if !fn.Accepts(1) {
		throw(c.At(1), "filter closure should expect exactly one argument, but it has %d", fn.Arity())
	}

	out := make([]Value, 0, len(arr))

	for i, v := range arr {
		isTruthy, ok := Truthy(in.callFnValue(fn, c.Site, fnLabel(fn), []Value{v}))

		if !ok {
			throw(c.At(1), "failed when applying closure to array value at index %d of type %s. the closure didn't returned a truthy value", i, v.Kind())
		}

		if isTruthy {
			out = append(out, v)
		}
	}

	filtered := Array{
		Items: out,
	}

	return &filtered
}

func builtinMap(in *Interp, c *Call) Value {
	arr := expectKindAt(c.At(0), c.Args[0], KArray).(*Array).Items
	fn := expectKindAt(c.At(1), c.Args[1], KFn).(*FnValue)

	if !fn.Accepts(1) {
		throw(c.At(1), "map closure should expect exactly one argument, but it has %d", fn.Arity())
	}

	out := make([]Value, len(arr))

	for i, v := range arr {
		value := in.callFnValue(fn, c.Site, fnLabel(fn), []Value{v})

		out[i] = value
	}

	mapped := Array{
		Items: out,
	}

	return &mapped
}

func builtinAppend(in *Interp, c *Call) Value {
	arrVal := c.Args[0]

	arr, ok := arrVal.(*Array)
	if !ok {
		throw(c.At(0), "first argument to append must be an array")
	}

	val := c.Args[1]
	arr.Items = append(arr.Items, val)

	return arrVal
}

// builtinDelete implements delete(array, start, end?): removes the single
// index start, or the range [start, end) when end is given -- same
// half-open convention as select()'s [from, to). Mutates array in place and
// returns it.
func builtinDelete(in *Interp, c *Call) Value {
	if c.N() > 3 {
		throw(c.Site, "too many arguments delete(...). expected 2 or 3 but got %d", c.N())
	} else if c.N() < 2 {
		throw(c.Site, "too few arguments delete(...). expected 2 or 3 but got %d", c.N())
	}

	arr := expectKindAt(c.At(0), c.Args[0], KArray).(*Array)
	start := intOf(expectKindAt(c.At(1), c.Args[1], KInt))

	length := int64(len(arr.Items))

	end := start + 1
	if c.N() == 3 {
		end = intOf(expectKindAt(c.At(2), c.Args[2], KInt))
	}

	if start < 0 || start >= length {
		throw(c.At(1), "start '%d' is out of range. The size of the array is %d", start, length)
	}
	if end < 0 || end > length {
		throw(c.At(c.N()-1), "end '%d' is out of range. The size of the array is %d", end, length)
	}
	if start > end {
		throw(c.At(1), "start '%d' cannot be greater than end '%d'", start, end)
	}

	arr.Items = append(arr.Items[:start], arr.Items[end:]...)

	return ArrayV(arr)
}
