package random

import (
	"testing"

	"mca/internal/packages/pkgtest"
)

func r(src string) string { return "const r = import('random')\n" + src }

func TestSeededSequenceIsDeterministic(t *testing.T) {
	pkgtest.CheckUnit(t, r("r.srand(4)"))
	// glibc-compatible RNG, verified against the C binary
	pkgtest.CheckInt(t, r("r.srand(4); r.rand(1, 10)"), 2)

	// re-seeding restarts the sequence
	pkgtest.CheckBool(t, r("r.srand(4); a = r.rand(1, 10); r.srand(4); b = r.rand(1, 10); a == b"), true)
}

func TestRandStaysInRange(t *testing.T) {
	pkgtest.CheckBool(t, r("r.srand(1); ok = true; for i : 100 { v = r.rand(3, 7); if v < 3 or v > 7 { ok = false } }; ok"), true)
	pkgtest.CheckInt(t, r("r.rand(5, 5)"), 5) // single-value range
}

func TestErrors(t *testing.T) {
	pkgtest.ExpectError(t, r("r.rand(10, 1)"), "min (10) cannot be greater than max (1)")
	pkgtest.ExpectError(t, r("r.rand(1.5, 2)"), "unexpected data type")
	pkgtest.ExpectError(t, r("r.srand('4')"), "unexpected data type")
	pkgtest.ExpectError(t, r("r.rand(1)"), "")
	pkgtest.ExpectError(t, r("r.srand()"), "")
}
