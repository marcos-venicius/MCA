// Package io is MCA's `io` package, reached with `const io = import('io')`.
// It holds the file-touching functions; print, println and exit are everyday
// language furniture and stayed behind as builtins.
package io

import (
	"os"
	"path/filepath"
	"sync"

	"mca/internal/interp"
)

// openFiles tracks the files io.open has handed out and io.close has not yet
// reclaimed. MCA has no file value type, so a program holds an open file as an
// opaque integer handle and every io call routes back through this table. The
// handle is our own counter, not the OS file descriptor, so a use-after-close
// names a missing entry -- a clean runtime error -- instead of whatever the
// descriptor was recycled into. The mutex guards the two globals since this is
// package-level shared state.
var (
	openFilesMu sync.Mutex
	openFiles   = map[int64]*os.File{}
	nextHandle  int64 = 1
)

func init() {
	interp.RegisterModule(&interp.Module{
		Name: "io",
		Fns: map[string]*interp.Native{
			"read_entire_file": interp.NewNative("io.read_entire_file", 1, readEntireFile),
			"abspath":          interp.NewNative("io.abspath", 1, absPath),
			"open":             interp.NewNative("io.open", 3, openFile),
			"write":            interp.NewNative("io.write", 2, writeFile),
			"close":            interp.NewNative("io.close", 1, closeFile),
		},
		Constants: map[string]interp.Value{
			"O_RDONLY": interp.IntV(int64(os.O_RDONLY)),
			"O_WRONLY": interp.IntV(int64(os.O_WRONLY)),
			"O_RDWR":   interp.IntV(int64(os.O_RDWR)),

			"O_APPEND": interp.IntV(int64(os.O_APPEND)),
			"O_CREATE": interp.IntV(int64(os.O_CREATE)),
			"O_EXCL":   interp.IntV(int64(os.O_EXCL)),
			"O_SYNC":   interp.IntV(int64(os.O_SYNC)),
			"O_TRUNC":  interp.IntV(int64(os.O_TRUNC)),

			// Permission bits for io.open's mode argument -- the low nine bits
			// of a Unix file mode, OR'd together. Portable octal literals
			// rather than syscall.S_* so the values are the same everywhere.
			"S_IRUSR": interp.IntV(0o400),
			"S_IWUSR": interp.IntV(0o200),
			"S_IXUSR": interp.IntV(0o100),
			"S_IRWXU": interp.IntV(0o700),

			"S_IRGRP": interp.IntV(0o040),
			"S_IWGRP": interp.IntV(0o020),
			"S_IXGRP": interp.IntV(0o010),
			"S_IRWXG": interp.IntV(0o070),

			"S_IROTH": interp.IntV(0o004),
			"S_IWOTH": interp.IntV(0o002),
			"S_IXOTH": interp.IntV(0o001),
			"S_IRWXO": interp.IntV(0o007),
		},
		// One-line gloss for each constant, shown by help('io') and
		// help('io.<name>'). Same keys as Constants above.
		ConstantDocs: map[string]string{
			"O_RDONLY": "open the file read-only.",
			"O_WRONLY": "open the file write-only.",
			"O_RDWR":   "open the file read-write.",

			"O_APPEND": "append data to the file when writing.",
			"O_CREATE": "create a new file if none exists.",
			"O_EXCL":   "used with O_CREATE, the file must not already exist.",
			"O_SYNC":   "open for synchronous I/O.",
			"O_TRUNC":  "truncate a regular writable file when opened.",

			"S_IRUSR": "owner: read.",
			"S_IWUSR": "owner: write.",
			"S_IXUSR": "owner: execute.",
			"S_IRWXU": "owner: read, write and execute.",

			"S_IRGRP": "group: read.",
			"S_IWGRP": "group: write.",
			"S_IXGRP": "group: execute.",
			"S_IRWXG": "group: read, write and execute.",

			"S_IROTH": "others: read.",
			"S_IWOTH": "others: write.",
			"S_IXOTH": "others: execute.",
			"S_IRWXO": "others: read, write and execute.",
		},
		Docs: map[string]interp.Doc{
			"read_entire_file": {
				Params:      []interp.Param{{Name: "path", Type: "string"}},
				Returns:     "string",
				Description: "Reads the whole contents of the file at path and returns it as a string. path resolves exactly like import()'s: a path starting with '.' is relative to the calling file's own directory, anything else is used as given. Throws a runtime error if the file cannot be read.",
				Examples:    []string{`io.read_entire_file('./data.txt')`},
			},
			"abspath": {
				Params:      []interp.Param{{Name: "path", Type: "string"}},
				Returns:     "string",
				Description: "Returns the absolute path of the file or folder at path, without requiring it to exist. path resolves exactly like import()'s: a path starting with '.' is relative to the calling file's own directory, anything else is taken as given and, when relative, resolved against the working directory. The result is cleaned (no '.', '..' or repeated separators). Throws a runtime error if the absolute path cannot be determined.",
				Examples:    []string{`io.abspath('./data.txt')`},
			},
			"open": {
				Params:      []interp.Param{{Name: "path", Type: "string"}, {Name: "flags", Type: "int"}, {Name: "mode", Type: "int"}},
				Returns:     "int",
				Description: "Opens the file at path and returns an integer handle to pass to io.write and io.close. flags is a combination of the io.O_* constants OR'd together (io.O_WRONLY | io.O_CREATE | io.O_TRUNC); mode is the permission bits used when the file is created, a combination of the io.S_I* constants (io.S_IRUSR | io.S_IWUSR | io.S_IRGRP | io.S_IROTH is 0644). path resolves like io.read_entire_file's. Throws a runtime error if the file cannot be opened. Every handle must be closed with io.close.",
				Examples:    []string{`fd = io.open('./out.txt', io.O_WRONLY | io.O_CREATE | io.O_TRUNC, io.S_IRUSR | io.S_IWUSR | io.S_IRGRP | io.S_IROTH)`},
			},
			"write": {
				Params:      []interp.Param{{Name: "handle", Type: "int"}, {Name: "text", Type: "string"}},
				Returns:     "unit",
				Description: "Writes text to the open file named by handle, exactly as given and with no trailing newline. handle must come from io.open and not yet be closed. Throws a runtime error if the handle is invalid or the write fails.",
				Examples:    []string{`io.write(fd, 'Hello, ')`},
			},
			"close": {
				Params:      []interp.Param{{Name: "handle", Type: "int"}},
				Returns:     "unit",
				Description: "Closes the open file named by handle and releases it; the handle is invalid afterwards. Throws a runtime error if the handle was already closed or never opened.",
				Examples:    []string{`io.close(fd)`},
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

// absPath resolves the path argument through the shared import()-style rules
// and then guarantees an absolute, cleaned result. ResolvePath already returns
// an absolute path for a '.'-prefixed path; for any other relative path it
// returns it unchanged, so filepath.Abs does the final work against the working
// directory. The file need not exist -- this is pure path arithmetic.
func absPath(in *interp.Interp, c *interp.Call) interp.Value {
	path := c.ResolvePath(c.StringArg(0))

	abs, err := filepath.Abs(path)
	if err != nil {
		interp.Throw(c.Site, "could not resolve absolute path of '%s': %s", path, err)
	}

	return interp.StringV(abs)
}

func openFile(in *interp.Interp, c *interp.Call) interp.Value {
	path := c.ResolvePath(c.StringArg(0))
	flag := c.IntArg(1)
	mode := c.IntArg(2)

	f, err := os.OpenFile(path, int(flag), os.FileMode(mode))
	if err != nil {
		interp.Throw(c.Site, "could not open file '%s': %s", path, err)
	}

	openFilesMu.Lock()
	handle := nextHandle
	nextHandle++
	openFiles[handle] = f
	openFilesMu.Unlock()

	return interp.IntV(handle)
}

// lookupFile resolves argument i, an int file handle, to the open file it
// names. A handle that io.close already reclaimed -- or that io.open never
// handed out -- is a runtime error rather than a nil dereference.
func lookupFile(c *interp.Call, i int) *os.File {
	handle := c.IntArg(i)

	openFilesMu.Lock()
	f, ok := openFiles[handle]
	openFilesMu.Unlock()

	if !ok {
		interp.Throw(c.At(i), "invalid file handle %d (already closed, or never opened)", handle)
	}

	return f
}

func writeFile(in *interp.Interp, c *interp.Call) interp.Value {
	f := lookupFile(c, 0)
	text := c.StringArg(1)

	if _, err := f.WriteString(text); err != nil {
		interp.Throw(c.Site, "could not write to file: %s", err)
	}

	return interp.UnitV()
}

func closeFile(in *interp.Interp, c *interp.Call) interp.Value {
	handle := c.IntArg(0)

	openFilesMu.Lock()
	f, ok := openFiles[handle]
	if ok {
		delete(openFiles, handle)
	}
	openFilesMu.Unlock()

	if !ok {
		interp.Throw(c.At(0), "invalid file handle %d (already closed, or never opened)", handle)
	}

	if err := f.Close(); err != nil {
		interp.Throw(c.Site, "could not close file: %s", err)
	}

	return interp.UnitV()
}
