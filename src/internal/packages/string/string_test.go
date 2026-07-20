package string

import (
	"strings"
	"testing"

	"mca/internal/packages/pkgtest"
)

// s prefixes src with the package import, so a test reads the way a program
// using the package does.
func s(src string) string { return "const s = import('string')\n" + src }

func TestRepeat(t *testing.T) {
	pkgtest.CheckString(t, s("s.repeat('ab', 3)"), "ababab")
	pkgtest.CheckString(t, s("s.repeat('a', 1)"), "a") // count of 1 -> unchanged
	pkgtest.CheckString(t, s("s.repeat('a', 0)"), "")  // count of 0 -> empty string
	pkgtest.CheckString(t, s("s.repeat('', 5)"), "")   // empty input -> empty regardless of count
	pkgtest.CheckString(t, s("s.repeat('xy', 4)"), "xyxyxyxy")

	pkgtest.ExpectError(t, s("s.repeat('a', -1)"), "cannot be negative")
	pkgtest.ExpectError(t, s("s.repeat(123, 3)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.repeat('a', 'b')"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.repeat('a', 1.5)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.repeat('a')"), "")
	pkgtest.ExpectError(t, s("s.repeat('a', 2, 3)"), "")
}

func TestReplace(t *testing.T) {
	pkgtest.CheckString(t, s("s.replace('Hello World', 'World', 'There')"), "Hello There")
	pkgtest.CheckString(t, s("s.replace('aaa', 'a', 'b')"), "bbb")                            // all occurrences replaced
	pkgtest.CheckString(t, s("s.replace('Hello World', 'x', 'y')"), "Hello World")            // no match -> unchanged
	pkgtest.CheckString(t, s("s.replace('Hello World', '', 'x')"), "xHxexlxlxox xWxoxrxlxdx") // empty old -- matches between every rune, matching strings.ReplaceAll
	pkgtest.CheckString(t, s("s.replace('Hello World', 'World', '')"), "Hello ")              // empty new -- deletes matches
	pkgtest.CheckString(t, s("s.replace('', 'a', 'b')"), "")                                  // empty haystack
	pkgtest.CheckString(t, s("s.replace('aaaa', 'aa', 'b')"), "bb")                           // matches consumed left to right, non-overlapping
	pkgtest.CheckString(t, s("s.replace('Hello', 'hello', 'x')"), "Hello")                    // case-sensitive

	pkgtest.ExpectError(t, s("s.replace(123, 'a', 'b')"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.replace('a', 123, 'b')"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.replace('a', 'a', 123)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.replace('a', 'b')"), "")
	pkgtest.ExpectError(t, s("s.replace('a', 'b', 'c', 'd')"), "")
}

func TestStartsWith(t *testing.T) {
	pkgtest.CheckBool(t, s("s.starts_with('Hello World', 'Hello')"), true)
	pkgtest.CheckBool(t, s("s.starts_with('Hello World', 'World')"), false)
	pkgtest.CheckBool(t, s("s.starts_with('Hello World', '')"), true)       // empty prefix always matches
	pkgtest.CheckBool(t, s("s.starts_with('', 'a')"), false)                // empty haystack, non-empty prefix
	pkgtest.CheckBool(t, s("s.starts_with('Hello', 'Hello World')"), false) // prefix longer than string
	pkgtest.CheckBool(t, s("s.starts_with('hello', 'Hello')"), false)       // case-sensitive

	pkgtest.ExpectError(t, s("s.starts_with(123, 'a')"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.starts_with('a', 123)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.starts_with('a')"), "")
	pkgtest.ExpectError(t, s("s.starts_with('a', 'b', 'c')"), "")
}

func TestEndsWith(t *testing.T) {
	pkgtest.CheckBool(t, s("s.ends_with('Hello World', 'World')"), true)
	pkgtest.CheckBool(t, s("s.ends_with('Hello World', 'Hello')"), false)
	pkgtest.CheckBool(t, s("s.ends_with('Hello World', '')"), true)       // empty suffix always matches
	pkgtest.CheckBool(t, s("s.ends_with('', 'a')"), false)                // empty haystack, non-empty suffix
	pkgtest.CheckBool(t, s("s.ends_with('World', 'Hello World')"), false) // suffix longer than string
	pkgtest.CheckBool(t, s("s.ends_with('world', 'World')"), false)       // case-sensitive

	pkgtest.ExpectError(t, s("s.ends_with(123, 'a')"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.ends_with('a')"), "")
}

func TestLowerUpper(t *testing.T) {
	pkgtest.CheckString(t, s("s.lower('HELLO')"), "hello")
	pkgtest.CheckString(t, s("s.lower('MiXeD123!')"), "mixed123!")
	pkgtest.CheckString(t, s("s.lower('')"), "")
	pkgtest.CheckString(t, s("s.upper('hello')"), "HELLO")
	pkgtest.CheckString(t, s("s.upper('MiXeD123!')"), "MIXED123!")
	pkgtest.CheckString(t, s("s.upper('')"), "")

	pkgtest.ExpectError(t, s("s.lower(123)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.upper(123)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.lower()"), "")
	pkgtest.ExpectError(t, s("s.upper('a', 'b')"), "")
}

func TestTrims(t *testing.T) {
	pkgtest.CheckString(t, s("s.trim('  hello  ')"), "hello")
	pkgtest.CheckString(t, s("s.trim('   ')"), "")
	pkgtest.CheckString(t, s("s.trim('  hello   world  ')"), "hello   world") // interior spaces preserved
	pkgtest.CheckString(t, s("s.ltrim('  hello  ')"), "hello  ")              // only leading whitespace removed
	pkgtest.CheckString(t, s("s.rtrim('  hello  ')"), "  hello")              // only trailing whitespace removed

	pkgtest.ExpectError(t, s("s.trim(123)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.ltrim(true)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.rtrim(['a'])"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.trim()"), "")
}

func TestJoin(t *testing.T) {
	pkgtest.CheckString(t, s("s.join(['a', 'b', 'c'], ',')"), "a,b,c")
	pkgtest.CheckString(t, s("s.join(['a', 'b', 'c'], '')"), "abc")
	pkgtest.CheckString(t, s("s.join(['a', 'b', 'c'], ' -- ')"), "a -- b -- c")
	pkgtest.CheckString(t, s("s.join(['solo'], ',')"), "solo")
	pkgtest.CheckString(t, s("s.join([], ',')"), "")

	pkgtest.ExpectError(t, s("s.join([1, 2, 3], ',')"), "expected a string at index 0")
	pkgtest.ExpectError(t, s("s.join(['a', 2, 'c'], ',')"), "expected a string at index 1") // fails on the first non-string
	pkgtest.ExpectError(t, s("s.join('not an array', ',')"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.join(['a', 'b'], 1)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.join(['a', 'b'])"), "")
}

func TestSplit(t *testing.T) {
	pkgtest.CheckInt(t, s("len(s.split('a,b,c', ','))"), 3)
	pkgtest.CheckString(t, s("s.split('a,b,c', ',')[1]"), "b")
	pkgtest.CheckString(t, s("s.join(s.split('a,b,c', ','), ',')"), "a,b,c") // round-trips through join

	pkgtest.CheckString(t, s("s.split('a::b::c', '::')[1]"), "b") // multi-char separator

	pkgtest.CheckInt(t, s("len(s.split('hello', ','))"), 1) // separator not present -> whole string, unsplit
	pkgtest.CheckString(t, s("s.split('hello', ',')[0]"), "hello")

	pkgtest.CheckInt(t, s("len(s.split('', ','))"), 1) // empty input -> single empty-string element
	pkgtest.CheckString(t, s("s.split('', ',')[0]"), "")

	pkgtest.CheckInt(t, s("len(s.split('a,,b', ','))"), 3) // consecutive separators -> empty element between them
	pkgtest.CheckString(t, s("s.split('a,,b', ',')[1]"), "")

	pkgtest.CheckInt(t, s("len(s.split(',a,', ','))"), 3) // leading/trailing separators -> empty elements at the ends
	pkgtest.CheckString(t, s("s.split(',a,', ',')[0]"), "")

	pkgtest.CheckInt(t, s("len(s.split('abc', ''))"), 3) // empty separator -> split between every rune
	pkgtest.CheckString(t, s("s.split('abc', '')[0]"), "a")

	pkgtest.ExpectError(t, s("s.split(123, ',')"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.split('a,b', 123)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.split('a,b')"), "")
}

func TestOrdChr(t *testing.T) {
	pkgtest.CheckInt(t, s("s.ord('a')"), int64('a'))
	pkgtest.CheckInt(t, s("s.ord('z')"), int64('z'))
	pkgtest.CheckString(t, s("s.chr(65)"), "A")
	pkgtest.CheckString(t, s("s.chr(32)"), " ")

	pkgtest.CheckString(t, s("s.chr(128512)"), "\U0001F600") // multi-byte code point (an emoji)
	pkgtest.CheckInt(t, s("len(s.chr(128512))"), 4)          // ... encoded as 4 UTF-8 bytes

	// round-trips through ord() for single-byte code points (ord() indexes
	// by byte, not rune, so this doesn't hold for multi-byte code points).
	pkgtest.CheckInt(t, s("s.ord(s.chr(65))"), 65)
	pkgtest.CheckString(t, s("s.chr(s.ord('Q'))"), "Q")

	// Out-of-range code points fall back to the Unicode replacement
	// character (U+FFFD) rather than erroring, matching Go's string(rune(...)).
	pkgtest.CheckString(t, s("s.chr(-1)"), "�")
	pkgtest.CheckString(t, s("s.chr(2000000)"), "�")

	pkgtest.ExpectError(t, s("s.ord('ab')"), "length 1")
	pkgtest.ExpectError(t, s("s.ord(97)"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.chr('a')"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.chr(1.5)"), "unexpected data type")
}

func TestFormat(t *testing.T) {
	pkgtest.CheckString(t,
		s("s.format('Hello ', 'World!', ' I am ', 5, ' years old. And ', 5.6, ' feet. I am a ', true, ' tall. I am not ', false)"),
		"Hello World! I am 5 years old. And 5.6 feet. I am a true tall. I am not false")
	pkgtest.CheckString(t, s("s.format(42)"), "42")

	pkgtest.ExpectError(t, s("s.format()"), "at least one argument")
	pkgtest.ExpectError(t, s("s.format([1])"), "unexpected data type")
	pkgtest.ExpectError(t, s("s.format(?)"), "unexpected data type")
}

// A package function is an ordinary value: storable, passable to the
// higher-order builtins, arity-checked when called through a value.
func TestFunctionsAreFirstClass(t *testing.T) {
	pkgtest.CheckString(t, s("f = s.upper; f('hi')"), "HI")
	pkgtest.CheckString(t, s("fns = [s.lower, s.upper]; fns[1]('hi')"), "HI")
	pkgtest.CheckString(t, s("s.join(map(['a', 'b'], s.upper), '')"), "AB")
	pkgtest.CheckInt(t, s("len(map([1, 2], s.format))"), 2) // variadic works as a fixed-arity callback
	pkgtest.ExpectError(t, s("f = s.upper; f()"), "")
	pkgtest.ExpectError(t, s("f = s.upper; f('a', 'b')"), "")
}

func TestHelp(t *testing.T) {
	for _, src := range []string{"help('string')", "help('string.upper')", "help(import('string').upper)"} {
		_, out, err := pkgtest.Run(t, src)
		if err != nil {
			t.Fatalf("%q: unexpected runtime error: %v", src, err)
		}

		if !strings.Contains(out, "string.upper(str: string) -> string") {
			t.Errorf("%q: expected the signature of string.upper, got:\n%s", src, out)
		}
	}
}
