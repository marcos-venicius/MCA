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
- **Map**: Key-value data structure. Keys are always `int` or `string`; values may be `int`, `float`, `bool`, `string`, or a function — **not** another array or map (maps can't nest arrays/maps directly as values).
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

### Modules

`import(path)` reads, parses, and runs another `.mca` file in a fresh, isolated environment, returning that file's last top-level expression — by convention, a map of "exported" functions/values:

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
- [`type-casting.mca`](./examples/type-casting.mca), [`type-inspect.mca`](./examples/type-inspect.mca) — `as_*`/`is_*`/`type()`

## 3. Standard Library

MCA is bundled with built-in functions covering mathematics, strings, arrays, maps, and system utilities. All are called like any other function (e.g. `PI()`, not a bare identifier).

### Type Checking, Casting, and Introspection
- **`type(x)`**: returns the type name as a string — one of `'unit'`, `'int'`, `'float'`, `'bool'`, `'string'`, `'array'`, `'map'`, `'fn'`.
- **`is_int(x)`**, **`is_float(x)`**, **`is_bool(x)`**, **`is_string(x)`**, **`is_arr(x)`**, **`is_map(x)`**, **`is_fn(x)`**, **`is_unit(x)`**
- **`as_int(x)`**, **`as_float(x)`**, **`as_string(x)`** — cast between int/float/bool/string; accept numeric-looking strings and raise a runtime error on invalid input.
- **`as_bool(x)`** — follows the [truthiness](#truthiness) rules above (so `as_bool('')` is `false`, `as_bool('x')` is `true` — it's not a `'true'`/`'false'` string parse).

### Strings
- **`len(s)`**: length (also works on arrays and maps).
- `s[index]`: indexing returns a 1-character string (0-based, no negative indices).
- **`select(s, from, to)`**: substring from `from` (inclusive) to `to` (exclusive).
- **`ord(s)`**: ASCII code of a 1-character string.
- **`format(a, b, ...)`**: concatenates any mix of int/float/bool/string arguments into one string (floats use up to 6 significant digits, unlike `as_string`'s fixed 6 decimal places).

### Arrays
- **`len(a)`**, **`append(a, value)`** (mutates `a` in place and returns it), `a[index] = value` (mutation), `a[index]` (0-based, no negative indices).

### Maps
- Construct with `{ 'k': v, ... }`; read/write with `m[key]`, `m.field`, or call a stored function with `m.method(args)`.
- **`len(m)`**, **`map_del(m, key)`** (returns whether the key existed), **`map_clear(m)`**.
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
- **`import(path)`**: load and run another `.mca` file; see [Modules](#modules) above.
- **`time()`**: Unix timestamp in seconds. **`millisecond()`**: current time in milliseconds.
- **Date Utilities**, each taking an integer *hour offset* from now (in UTC): `year(offset)`, `month(offset)`, `date(offset)`, `day(offset)` (0=Sunday..6=Saturday), `hour(offset)`, `minute(offset)`, `second(offset)`. Pass `0` for "now".

## 4. Language Caveats

1. **Mandatory Parentheses**: You cannot reference a function without calling it unless you are intentionally passing it by reference. For zero-argument function invocations, you must use parentheses (e.g., `time()` or `PI()`).
2. **Calls only work on bare names**: `f(x)` is only recognized as a call when `f` is a plain identifier immediately followed by `(` — an arbitrary expression followed by `(` (e.g. `arr[0](5)`) does *not* parse as a call. The one exception is the map `m.method(args)` sugar, which is handled specially by the indexing grammar.
3. **Semicolons and newlines**: statements implicitly return the value of their last expression, and newlines carry no special meaning (no automatic semicolon insertion) — most statement sequences resolve fine without explicit `;`. One notable exception: a bare `break`/`return` immediately followed by `}` (with nothing else on the line) needs an explicit `;` before the `}`, since the parser otherwise tries to parse a value expression after `break`/`return`.
4. **Strings only support `==`/`!=`**: any other binary operator between two strings, or between a string and a non-string, raises a runtime error — there's no implicit numeric coercion for strings.
5. **Unit (`?`) only supports `==`/`!=` as a binary operator**: using `?` with any other operator (`+`, `<`, ...) raises a runtime error. It does have a defined truthiness though — always falsy — so `if x { ... }` works fine when `x` is `?`.
6. **No negative indexing**: `a[-1]` / `s[-1]` are out-of-bounds errors, not "last element" access.
7. **Map values can't be arrays or maps**: only `int`/`float`/`bool`/`string`/function values can be stored in a map (see [Data Types](#data-types)).
8. **Numeric literals**: decimal only — no hex/binary/exponent notation, no digit-group separators.
9. **`and`/`or` are stricter than `if`/`while`**: they only accept `bool`/`int`/`float` operands, unlike the wider [truthiness](#truthiness) rules `if`/`while`/`as_bool()` use.
10. **Map iteration order is unspecified**: don't write code that depends on the order `for k, v : m` or `println(m)` visits keys in.

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
