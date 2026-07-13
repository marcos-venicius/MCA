// Package crypt is MCA's `crypt` package: hashing and digests, reached from a
// program with `const crypt = import('crypt')`.
//
// It is the first native package, so it is also the template for the next
// one. A package is a Go package of its own under internal/packages, it
// registers itself from init(), and it depends on interp -- interp never
// depends on it. internal/packages blank-imports this one to make the init()
// run; that is the only edit outside this directory a new package needs.
//
// MCA has no bytes type, so a digest crosses back into the language as a
// lowercase hex string.
package crypt

import (
	"crypto/md5"
	"encoding/hex"

	"mca/internal/interp"
)

func init() {
	interp.RegisterModule(&interp.Module{
		Name: "crypt",
		Fns: map[string]*interp.Native{
			"md5": interp.NewNative("crypt.md5", 1, md5Hex),
		},
		Docs: map[string]interp.Doc{
			"md5": {
				Params:      []interp.Param{{Name: "s", Type: "string"}},
				Returns:     "string",
				Description: "MD5 digest of s, as a 32-character lowercase hex string. MD5 is broken as a cryptographic hash -- fine for checksums and cache keys, not for signatures or passwords.",
				Examples: []string{
					"crypt.md5('hello')  -- '5d41402abc4b2a76b9719d911017c592'",
				},
			},
		},
	})
}

func md5Hex(in *interp.Interp, c *interp.Call) interp.Value {
	sum := md5.Sum([]byte(c.StringArg(0)))
	return interp.StringV(hex.EncodeToString(sum[:]))
}
