package interp

import "mca/internal/ast"

// glibcRand reimplements glibc's default rand()/srand() algorithm exactly
// (the TYPE_3 additive-feedback generator, degree 31 / separation 3,
// originally from 4.3BSD random.c) so that seeded sequences are bit-for-bit
// identical to the C build's -- verified empirically against the compiled
// C binary across several seeds during development (plan decision 6: this
// worked out, so no test/behavior had to be changed).
//
// C's rand()/srand() are genuine libc *process-global* state, not per
// M_Interpreter -- so this is deliberately a package-level variable rather
// than a field on Interp, matching that global-ness (e.g. a module loaded
// via import() shares the same rand stream continuation as its importer,
// exactly as it would sharing libc's global state in C).
var globalRand = newGlibcRand(1) // glibc's default pre-seeded state uses seed 1

const randDeg = 31
const randSep = 3

type glibcRand struct {
	table [randDeg]int32
	fptr  int
	rptr  int
}

func newGlibcRand(seed uint32) *glibcRand {
	r := &glibcRand{}
	r.seed(seed)
	return r
}

func (r *glibcRand) seed(seed uint32) {
	if seed == 0 {
		seed = 1
	}

	word := int32(seed)
	r.table[0] = word

	// TODO: to be honest, I don't understand exactly why those calculations
	//       just internet stuff (should check it out again)
	for i := 1; i < randDeg; i++ {
		hi := word / 127773
		lo := word % 127773
		w := 16807*lo - 2836*hi
		if w < 0 {
			w += 2147483647
		}
		word = w
		r.table[i] = word
	}

	r.fptr = randSep
	r.rptr = 0

	for range randDeg * 10 {
		r.next()
	}
}

func (r *glibcRand) next() int32 {
	r.table[r.fptr] += r.table[r.rptr]
	result := int32(uint32(r.table[r.fptr]) >> 1)

	r.fptr++
	if r.fptr >= randDeg {
		r.fptr = 0
	}
	r.rptr++
	if r.rptr >= randDeg {
		r.rptr = 0
	}

	return result
}

func builtinSrand(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	seed := intOf(expectKind(args[0], in.Eval(args[0]).Value, KInt))
	globalRand.seed(uint32(seed))
	return UnitV()
}

func builtinRand(in *Interp, caller ast.Expr, args []ast.Expr) Value {
	min := intOf(expectKind(args[0], in.Eval(args[0]).Value, KInt))
	max := intOf(expectKind(args[1], in.Eval(args[1]).Value, KInt))

	if min > max {
		throw(args[0].Pos(), "invalid range for rand(). min (%d) cannot be greater than max (%d)", min, max)
	}

	r := int64(globalRand.next())%(max-min+1) + min

	return IntV(r)
}
