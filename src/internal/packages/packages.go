// Package packages links MCA's native packages into the binary. It has no API
// of its own: importing it for effect runs each package's init(), which is
// what makes that package's name resolvable by import().
//
// Every native package is blank-imported here, and nowhere else. cmd/mca
// imports this one, so adding a package means adding its directory under
// internal/packages and one line below -- and nothing in interp changes.
//
// The indirection is what keeps the dependencies acyclic: a package imports
// interp (for Module, NewNative, Value, ...), so interp cannot import it back
// to register it. Somebody above both has to, and this is that somebody.
package packages

import (
	_ "mca/internal/packages/crypt"
	_ "mca/internal/packages/io"
	_ "mca/internal/packages/math"
	_ "mca/internal/packages/random"
	_ "mca/internal/packages/string"
)
