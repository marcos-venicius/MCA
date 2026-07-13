# MCA Language (Mere Computer Algorithm)

MCA is a dynamic expression-oriented toy scripting language. Originally born as a math calculator, it has evolved into a fully-fledged, expression-centric scripting language featuring functions, closures, dynamic data types, arrays, maps, and a small standard library. In MCA, almost every construct evaluates to a value.

<img width="1924" height="1050" alt="image" src="https://github.com/user-attachments/assets/6a5277b1-28b5-48ce-ad45-70655aff3032" />

The implementation lives under [`src/`](./src/) and is written in Go. An earlier C implementation exists for reference on the `c-version-deprecated` branch, but `main` is Go-only going forward.

## 1. Key Features

### Expression-Oriented
Everything in MCA resolves to a value. Blocks `{ ... }` and control flow constructs (`if`, `while`, `for`, `break`) all implicitly evaluate to their last executed expression:

```r
x = if 10 > 5 { 'big' } else { 'small' }   # x == 'big'
y = while (n -= 1) > 0 { n }               # y == the last value of n before the loop stopped
```

### Data Types
MCA supports dynamic data typing with automatic coercion when performing mathematical operations (e.g., integer division with a remainder cleanly coerces to a Float).
- **Unit**: Represents nothing/empty value (`?`). Only supports `==`/`!=` as a binary operator; always considered "falsy" (see [Truthiness](#truthiness) below).
- **Integer**: 64-bit signed integer.
- **Float**: 64-bit floating point number.
- **Boolean**: `true` or `false`.
- **String**: Single-quoted character sequences (e.g. `'hello world'`), with a small set of escapes: `\\`, `\'`, `\n`.
- **Array**: Ordered, mutable, growable list of any values (including other arrays and maps).
- **Map**: Key-value data structure. Keys are always `int` or `string`; values may be any type, arrays and maps included.
- **Function**: First-class, closure-capturing, callable values.

### Truthiness

`if`/`elif`/`while` conditions (and the `as_bool()` cast) accept any value and apply these rules:
- `int`/`float`: truthy unless exactly `0`.
- `bool`: itself.
- `string`/`array`/`map`: truthy unless empty (`''`, `[]`, `{}` are falsy).
- `unit` (`?`): always falsy.

```r
if [] { println('unreachable') } else { println('empty array is falsy') }
if 'x' { println('non-empty string is truthy') }
println(as_bool(''), as_bool('x'), as_bool(?))   # false true false
```

`and`/`or` are stricter: they only accept `bool`/`int`/`float` operands and always short-circuit to a `bool` result — passing a string/array/map/unit to `and`/`or` is a runtime error, even though the same value would be fine in an `if`.

### Functions and Closures
Functions are first-class citizens in MCA. You can define anonymous functions and assign them to variables, pass them as arguments to other functions, and use closures to capture lexical scope.

Anonymous function syntax uses the `\(args...) -> body` notation:

```r
# A simple one-liner function
distance = \(x1, y1, x2, y2) -> sqrt((x1 - x2) ^ 2 + (y1 - y2) ^ 2)

# Multi-line function using block syntax
display = \(message, formatter) -> {
    # 'formatter' is an anonymous function passed as a parameter
    println('[LOG] ', formatter(message))
}

# Passing an anonymous function as a parameter
display('Hello World', \(m) -> format('<', m, '>'))
```

Closures capture the environment they were defined in, and each call to a function that returns a new closure gets its own independent captured state:

```r
make_counter = \() -> {
    n = 0
    \() -> (n += 1)
}

c1 = make_counter()
c2 = make_counter()

println(c1(), c1(), c1())   # 1 2 3
println(c2())                # 1 -- c2 has its own independent 'n'
```

### Constants

`const name = value` binds a name that can never be reassigned. Constness belongs to the *binding*, not to the value, so `const a = [1]` freezes the name `a` while leaving the array it points at mutable:

```r
const MAX = 10
MAX = 20            # runtime error: you cannot modify constant values

const nums = [1, 2]
append(nums, 3)     # fine -- the name is frozen, the array is not
```

Redeclaring a constant in the scope that already owns it is an error, but an inner scope may still shadow it with a binding of its own.

### Builtins Are Values

Builtins aren't a reserved side-table of names you may only write before a `(` — they're ordinary constant values living in a scope just beneath the global one. So a builtin can be stored, passed, and returned like anything else:

```r
f = sort
println(f([2, 1], \(a, b) -> a - b))    # [1, 2]

println(map(['a', 'b'], upper))          # ['A', 'B'] -- passed by name
```

Because that scope sits *below* the global one, assigning to a builtin's name shadows it for the assigning scope rather than overwriting it — every other scope still sees the builtin. This is what lets a program use `len`, `year` or `help` as an ordinary variable name without breaking the language for everyone else:

```r
len = 5           # shadows the builtin here; len() still means len() elsewhere
println(len)      # 5
```

### Control Flow

**`if` / `elif` / `else`** — every branch is a block (or a single bare expression), and the whole construct evaluates to whichever branch ran:

```r
grade = \(score) -> if score >= 90 'A'
                    elif score >= 80 'B'
                    elif score >= 70 'C'
                    else 'F'

println(grade(85))   # B
```

**`while`** — an optional condition plus a body; `break [value]` stops the loop early and supplies the loop's own value:

```r
n = 0
result = while true {
    n += 1
    if n == 5 { break n * 100 }
}
println(result)   # 500
```

**`for`** has two forms. Range form, over integers:

```r
for i : 5               { print(i, ' ') }   # 0 1 2 3 4         (equivalent to [0, 5))
for i : [2, 10]         { print(i, ' ') }   # 2 3 4 5 6 7 8 9
for i : [10, -1, -1]    { print(i, ' ') }   # 10 9 8 7 6 5 4 3 2 1 0  (descending, inclusive of 0)
```

And "for-of" form, iterating an array, string, or map (key/value pairs; for arrays and strings the key is the index):

```r
for i, v : ['a', 'b', 'c'] { print(i, ':', v, ' ') }   # 0:a 1:b 2:c

css = { 'color': 'red', 'z-index': 10 }
for key, value : css { println(key, '=', value) }
```

Both loop forms support `break [value]` the same way `while` does.

### Arrays

```r
nums = [1, 2, 3]
append(nums, 4)          # nums is now [1, 2, 3, 4] -- append mutates in place
println(nums[0], len(nums))

nums[0] = 100             # index assignment
matrix = [[1, 0], [0, 1]] # arrays can nest arrays/maps freely
```

### Maps

Map literals use `{ 'key': value, ... }`. Besides `m[key]` indexing, MCA sugars `m.field` and `m.method(args)` for string-keyed access and for calling a function stored in a map (handy for building simple "module"/"object" style values):

```r
person = { 'name': 'Ada', 'greet': \(self) -> format('Hi, I am ', self.name) }

println(person['name'])   # Ada, via [] indexing
println(person.name)      # Ada, via . property sugar
println(person.greet(person))

person.age = 36           # assignment through . sugar too
map_del(person, 'age')
map_clear(person)
```

Missing keys read as `?` (unit) rather than raising an error. Map iteration/printing order (`for k, v : m`, `println(m)`) is unspecified — don't rely on key order.

A key may be written **without** a `: value`, in which case it initializes to `?` (unit). This pre-shapes a map with the keys it is going to hold, before it holds them:

```r
m = { name, age, 'email' }   # {'name': ?, 'age': ?, 'email': ?}

m.name = 'Marcos'            # ordinary entries -- assigning overwrites the unit
m['age'] = 32

record = { id: 1, name, tags: ['new'] }   # both forms mix freely in one literal
```

Duplicate keys are not an error: entries are written left to right, so the **last one wins** and the map keeps a single entry for that key — the same rule as assigning `m[k] = v` twice. An identifier key and the string key of the same name are the same key, so the two forms collide with each other:

```r
{ 'a': 1, 'a': 2 }['a']   # 2      -- overwritten, len is 1
{ a: 1, 'a': 2 }['a']     # 2      -- 'a' and a are one key
{ a: 1, a }['a']          # ?      -- a trailing bare key resets it to unit
```

### Modules and Packages

`import(name)` resolves one of two things, decided purely by the shape of its argument.

**A path is a file.** `import(path)` reads, parses, and runs another `.mca` file in a fresh, isolated environment, returning that file's last top-level expression — by convention, a map of "exported" functions/values:

```r
# utils.mca
pub_is_digit = \(c) -> ord(c) >= ord('0') and ord(c) <= ord('9')

# 'pub_' prefix is just a convention of mine since we don't have any keywords

{ 'is_digit': pub_is_digit }   # last expression = what import() returns
```

```r
# main.mca
utils = import('./utils.mca')
println(utils.is_digit('7'))   # true
```

A leading `.` resolves relative to the *importing file's* own directory, so modules can `import()` each other regardless of the caller's working directory. See [`examples/module/`](./examples/module/) for a complete multi-file example.

**A bare name is a package.** Packages are builtins that ship with MCA but are *not* bound to any name until a program asks for them, which keeps the global scope small — a program that doesn't do crypto never sees `crypt`. Nothing is read from disk, and the value you get back is an ordinary map, so it behaves exactly like a file module:

```r
const crypt = import('crypt')

println(crypt.md5('hello'))              # 5d41402abc4b2a76b9719d911017c592
println(map(['a', 'b'], crypt.md5))      # a package function is just a value
```

The two cases never overlap, so adding a package can't change what an existing file import resolves to, and a file can't shadow a package:

| `import(...)` | resolves to |
| --- | --- |
| `'./lexer.mca'` | file, relative to the **importing file's** directory |
| `'/opt/mca/lexer.mca'` | file, absolute path |
| `'lib/lexer.mca'` | file, relative to the working directory |
| `'crypt'` | package — never looked for on disk |

Anything that isn't a path (no leading `.`, not absolute, no `.mca` suffix) is a package name, and importing one that doesn't exist is a runtime error. Run `help()` to list the available packages, `help('crypt')` for one package's functions, and `help('crypt.md5')` for a single function. Available packages:

- **`crypt`** — hashing and digests. `crypt.md5(s)` returns the MD5 digest of `s` as a 32-character lowercase hex string. (MD5 is broken as a cryptographic hash: fine for checksums and cache keys, not for signatures or passwords.)

## 2. Examples

**Pascal's Triangle Rendering**
A rich example showcasing closures, string manipulations, variables, and math:

```r
help = \(error) -> {
    program_name = argv(0)

    println('usage: ', program_name, ' <triangle-height>')
    println()
    println('    triangle height must be greater than 0 and a valid integer\n')
    println()
    println('    Just try the number 5, for example.')

    if error != ?  println('\nerror: ', error)

    exit(1)
}

pad                = \(padding, char) -> while ((padding -= 1) >= 0) print(char)
next_pascal_number = \(p, x, y) -> p * (y - x + 1) / x

if argc() != 2  help(?)

NUM_ROWS = as_int(argv(1))

if (NUM_ROWS <= 0)    help(format('error: invalid triangle height: ', NUM_ROWS))
if !is_int(NUM_ROWS)  help(format('error: invalid triangle height: ', NUM_ROWS, '. it should be an integer value.'))

n = NUM_ROWS - 1
k = if (n % 2 == 0) n / 2 else (n - 1) / 2
x = 0

biggest_value = 1
while ((x += 1) < k + 1) biggest_value = next_pascal_number(biggest_value, x, n)

biggest_value_len = len(as_string(biggest_value)) + 1
padding           = as_int(NUM_ROWS * biggest_value_len / 2)

println()

row_index = -1

while (row_index += 1) < NUM_ROWS {
    pad(padding + biggest_value_len - 1, ' ')
    print(1)

    p = 1
    x = 0

    while (x += 1) < row_index + 1 {
        p = next_pascal_number(p, x, row_index)
        pad(biggest_value_len - len(as_string(p)), ' ')
        print(p)
    }

    println()

    padding -= (biggest_value_len / 2)
}

println()
```

In fact, this code is present in the examples folder [here](./examples/pascals-triangle.mca).

More examples, each focused on a specific feature, live in [`examples/`](./examples/):
- [`arrays.mca`](./examples/arrays.mca) — nested arrays, recursive printing
- [`maps.mca`](./examples/maps.mca) — map mutation, `map_del`/`map_clear`, iteration
- [`loops.mca`](./examples/loops.mca) — every `for` shape, plus for-of over strings/arrays/maps
- [`user-defined-functions.mca`](./examples/user-defined-functions.mca) — closures over global/lexical scope, passing functions as arguments
- [`module/`](./examples/module/) — a multi-file program using `import()`
- [`crypt.mca`](./examples/crypt.mca) — importing a package, and `help()` on it
- [`type-casting.mca`](./examples/type-casting.mca), [`type-inspect.mca`](./examples/type-inspect.mca) — `as_*`/`is_*`/`type()`

## 3. Standard Library

MCA is bundled with built-in functions covering mathematics, strings, arrays, maps, and system utilities. They are always available — no import needed — and are [ordinary values](#builtins-are-values), though still called like any other function (e.g. `PI()`, not a bare identifier).

**`help()`** documents the whole library from inside the language: `help()` lists every builtin by category (plus the importable packages), and `help('sort')` or `help(sort)` prints one function's signature, description, and examples. It is the authoritative reference — this section is a summary.

### Type Checking, Casting, and Introspection
- **`type(x)`**: returns the type name as a string — one of `'unit'`, `'int'`, `'float'`, `'bool'`, `'string'`, `'array'`, `'map'`, `'fn'`.
- **`is_int(x)`**, **`is_float(x)`**, **`is_bool(x)`**, **`is_string(x)`**, **`is_array(x)`**, **`is_map(x)`**, **`is_fn(x)`**, **`is_unit(x)`**
- **`as_int(x)`**, **`as_float(x)`**, **`as_string(x)`** — cast between int/float/bool/string; accept numeric-looking strings and raise a runtime error on invalid input.
- **`as_bool(x)`** — follows the [truthiness](#truthiness) rules above (so `as_bool('')` is `false`, `as_bool('x')` is `true` — it's not a `'true'`/`'false'` string parse).

### Strings
- **`len(s)`**: length (also works on arrays and maps).
- `s[index]`: indexing returns a 1-character string (0-based, no negative indices).
- **`select(s, from, to)`**: substring from `from` (inclusive) to `to` (exclusive).
- **`ord(s)`**, **`chr(n)`**: ASCII code of a 1-character string, and back.
- **`format(a, b, ...)`**: concatenates any mix of int/float/bool/string arguments into one string (floats use up to 6 significant digits, unlike `as_string`'s fixed 6 decimal places).
- **Case & whitespace**: `upper(s)`, `lower(s)`, `trim(s)`, `ltrim(s)`, `rtrim(s)`.
- **Search & edit**: `starts_with(s, prefix)`, `ends_with(s, suffix)`, `replace(s, old, new)`, `repeat(s, n)`.
- **Split & join**: `split(s, sep)` → array, `join(arr, sep)` → string.

### Arrays
- **`len(a)`**, `a[index]` (0-based, no negative indices), `a[index] = value`.
- **`append(a, value)`**, **`delete(a, start)`** / **`delete(a, start, end)`** (half-open `[start, end)`, like `select`) — both mutate `a` in place and return it.
- **`sort(a, cmp)`**, **`reverse(a)`**, **`concat(a, b, ...)`**, **`map(a, fn)`**, **`filter(a, fn)`** — all return a *new* array. `cmp` takes two elements and returns an int: negative if the first sorts before the second, positive if after, zero if equal.
- **`contains(a, value)`**, **`sum(a)`**.
- **`indexes_to_keys(a, m)`**: builds a new map by picking elements out of `a` at the indexes named in `m` — `indexes_to_keys(['x', 'y', 'z'], {0: 'first', 2: 'third'})` is `{'first': 'x', 'third': 'z'}`.

### Maps
- Construct with `{ 'k': v, ... }` (or `{ k }` to [initialize a key to `?`](#maps)); read/write with `m[key]`, `m.field`, or call a stored function with `m.method(args)`.
- **`len(m)`**, **`keys(m)`**, **`values(m)`**, **`map_del(m, key)`** (returns whether the key existed), **`map_clear(m)`**, **`contains(m, key)`**.
- Iterate with `for key, value : m`. Missing keys read as `?` rather than erroring. Iteration order is unspecified.

### Mathematical Constants & Functions
- **Constants**: `PI()`, `E()`
- **Basic Math**: `abs(x)`, `floor(x)`, `ceil(x)`, `round(x)`, `sqrt(x)`, `exp(x)`, `log(x)`, `log10(x)`, `max(x, y, ...)`, `min(x, y, ...)`
- **Trigonometry**: `sin(x)`, `cos(x)`, `tan(x)`, `asin(x)`, `acos(x)`. Standard evaluation is in radians. Converters: `rad(x)`, `deg(x)`.
- **Random**: `srand(seed)`, `rand(min, max)` (inclusive on both ends).

### Environment & I/O
- **`print(...)`**, **`println(...)`**: write to stdout; both return their last argument (or `?` if called with none).
- **`read_entire_file(path)`**: read an entire file into a string.
- **`argc()`**, **`argv(index)`**: CLI argument count/access. `argv(0)` is the script path itself.
- **`exit(code)`**: abort execution immediately with a status code.
- **`import(name)`**: load a `.mca` file, or a package by bare name; see [Modules and Packages](#modules-and-packages) above.
- **`help(...)`**: documentation, see above.
- **`time()`**: Unix timestamp in seconds. **`millisecond()`**: current time in milliseconds.
- **Date Utilities**, each taking an integer *hour offset* from now (in UTC): `year(offset)`, `month(offset)`, `date(offset)`, `day(offset)` (0=Sunday..6=Saturday), `hour(offset)`, `minute(offset)`, `second(offset)`. Pass `0` for "now".

### Packages (imported on demand)

Unlike the builtins above, a package is only bound when a program imports it — see [Modules and Packages](#modules-and-packages).

- **`crypt`** — `crypt.md5(s)`: MD5 digest of `s`, as a 32-character lowercase hex string.

## 4. Language Caveats

1. **Mandatory Parentheses**: You cannot reference a function without calling it unless you are intentionally passing it by reference. For zero-argument function invocations, you must use parentheses (e.g., `time()` or `PI()`).
2. **Calls only work on bare names**: `f(x)` is only recognized as a call when `f` is a plain identifier immediately followed by `(` — an arbitrary expression followed by `(` (e.g. `arr[0](5)`) does *not* parse as a call. The one exception is the map `m.method(args)` sugar, which is handled specially by the indexing grammar.
3. **Semicolons and newlines**: statements implicitly return the value of their last expression, and newlines carry no special meaning (no automatic semicolon insertion) — most statement sequences resolve fine without explicit `;`. One notable exception: a bare `break`/`return` immediately followed by `}` (with nothing else on the line) needs an explicit `;` before the `}`, since the parser otherwise tries to parse a value expression after `break`/`return`.
4. **Strings only support `==`/`!=`**: any other binary operator between two strings, or between a string and a non-string, raises a runtime error — there's no implicit numeric coercion for strings.
5. **Unit (`?`) only supports `==`/`!=` as a binary operator**: using `?` with any other operator (`+`, `<`, ...) raises a runtime error. It does have a defined truthiness though — always falsy — so `if x { ... }` works fine when `x` is `?`.
6. **No negative indexing**: `a[-1]` / `s[-1]` are out-of-bounds errors, not "last element" access.
7. **Numeric literals**: decimal only — no hex/binary/exponent notation, no digit-group separators.
8. **`and`/`or` are stricter than `if`/`while`**: they only accept `bool`/`int`/`float` operands, unlike the wider [truthiness](#truthiness) rules `if`/`while`/`as_bool()` use.
9. **Map iteration order is unspecified**: don't write code that depends on the order `for k, v : m` or `println(m)` visits keys in.
10. **Duplicate map keys silently overwrite**: `{ 'a': 1, 'a': 2 }` is one entry holding `2`, not an error (see [Maps](#maps)).
11. **Imports are re-evaluated, not cached**: importing the same file twice runs it twice, in two isolated environments. Packages are cheaper (nothing is parsed), but each `import('crypt')` still hands back a fresh map.

## 5. Building and Running

### Compiling

Build `./bin/mca` from the `src/` Go module:
```bash
make
```

(this just runs `cd src && go build -o ../bin/mca cmd/mca/main.go` — see the [`Makefile`](./Makefile))

### Running tests

```bash
cd src
go test ./...
```

Or run a script directly, no separate build step:
```bash
cd src
go run ./cmd/mca <file> [argv...]
```

### Usage Help

```bash
./bin/mca <file> [argv...]

    -h                  show this help
```
