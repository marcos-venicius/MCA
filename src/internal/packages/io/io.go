// Package io is MCA's `io` package, reached with `const io = import('io')`.
// It holds the file-touching functions; print, println and exit are everyday
// language furniture and stayed behind as builtins.
package io

import (
	"os"

	"mca/internal/interp"
)

func init() {
	interp.RegisterModule(&interp.Module{
		Name: "io",
		Fns: map[string]*interp.Native{
			"read_entire_file": interp.NewNative("io.read_entire_file", 1, readEntireFile),
		},
		Docs: map[string]interp.Doc{
			"read_entire_file": {
				Params:      []interp.Param{{Name: "path", Type: "string"}},
				Returns:     "string",
				Description: "Reads the whole contents of the file at path and returns it as a string. path resolves exactly like import()'s: a path starting with '.' is relative to the calling file's own directory, anything else is used as given. Throws a runtime error if the file cannot be read.",
				Examples:    []string{`io.read_entire_file('./data.txt')`},
			},
		},
	})
}

func readEntireFile(in *interp.Interp, c *interp.Call) interp.Value {
	path := c.ResolvePath(c.StringArg(0))

	content, err := os.ReadFile(path)
	if err != nil {
		interp.Throw(c.Site, "could not read file '%s': %s", path, err)
	}

	return interp.StringV(string(content))
}
