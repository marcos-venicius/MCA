# MCA Language (Mere Computer Algorithm)

MCA is a dynamic expression-oriented toy scripting language. Originally born as a math calculator, it has evolved into a fully-fledged, expression-centric scripting language featuring functions, closures, dynamic data types, and more. In MCA, almost every construct evaluates to a value.

<img width="1907" height="905" alt="image" src="https://github.com/user-attachments/assets/a318b160-337a-4c10-9a54-3e36096e27a8" />

## 1. Key Features

### Expression-Oriented
Everything in MCA resolves to a value. Blocks `{ ... }` and control flow statements (`if`, `while`, `break`) all implicitly evaluate to their last executed expression.

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

### Data Types
MCA supports dynamic data typing with automatic coercion when performing mathematical operations (e.g., integer division with a remainder cleanly coerces to a Float).
- **Unit**: Represents nothing/empty value (`?`).
- **Integer**: 64-bit signed integer (`int64_t`).
- **Float**: 64-bit floating point number (`double`).
- **Boolean**: `true` or `false`.
- **String**: Sequences of characters (e.g. `'hello world'`).
- **Map**: Key-value data structures.
- **Function**: First-class callable functions.

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

if NUM_ROWS <= 0      help(format('error: invalid triangle height: ', NUM_ROWS))
if !is_int(NUM_ROWS)  help(format('error: invalid triangle height: ', NUM_ROWS, '. it should be an integer value.'))

n = NUM_ROWS - 1
k = if (n % 2 == 0) n / 2 else (n - 1) / 2
x = 0

biggest_value = 1
while ((x += 1) < k + 1)  biggest_value = next_pascal_number(biggest_value, x, n)

biggest_value_len = len(as_string(biggest_value)) + 1
padding           = as_int(NUM_ROWS * biggest_value_len / 2)

println()

row_index = 0; while row_index < NUM_ROWS {
    pad(padding, ' ')
    padding -= (biggest_value_len / 2)
    pad(biggest_value_len - 1, ' ')
    print(1)
    p = 1
    x = 1; while x < row_index + 1 {
        p = next_pascal_number(p, x, row_index)
        pad(biggest_value_len - len(as_string(p)), ' ')
        print(p)
        x += 1
    }
    row_index += 1
    println()
}

println()
```

In fact, this code is present in the examples folder [here](./examples/pascals-triangle.mca).

## 3. Standard Library

MCA is bundled with built-in functions covering mathematics, strings, maps, and system utilities.

### Core Utilities & Strings
- **Type Checking**: `is_int(x)`, `is_float(x)`, `is_bool(x)`, `is_string(x)`, `is_unit(x)`
- **Type Casting**: `as_int(x)`, `as_float(x)`, `as_bool(x)`, `as_string(x)`
- **`type(x)`**: Returns the numeric representation of the type.
- **Strings**: `len(s)` (length), `at(s, index)` (char at index), `select(s, from, to)` (substring slice), `ord(s)` (ASCII code), `format(fmt, ...)` (string formatting)

### Maps
Key-value mappings can be managed via:
- `map_init()`, `map_set(m, k, v)`, `map_get(m, k)`, `map_del(m, k)`, `map_clear(m)`
- **Iteration**: `map_it(m)`, `map_it_done(it)`, `map_it_next(it)`, `map_it_key(it)`, `map_it_value(it)`

### Mathematical Constants & Functions
- **Constants**: `PI()`, `E()`
- **Basic Math**: `abs(x)`, `floor(x)`, `ceil(x)`, `round(x)`, `sqrt(x)`, `exp(x)`, `log(x)`, `log10(x)`, `max(x, y, ...)`, `min(x, y, ...)`
- **Trigonometry**: `sin(x)`, `cos(x)`, `tan(x)`, `asin(x)`, `acos(x)`. Standard evaluation is in radians. Converters: `rad(x)`, `deg(x)`.

### Environment & I/O
- **`print(...)`**, **`println(...)`**: Display to stdout.
- **`read_entire_file(path)`**: Read an entire file into a string.
- **`argc()`**, **`argv(index)`**: CLI arguments parsing.
- **`exit(code)`**: Abort execution with a status code.
- **`time()`**: Unix timestamp in seconds.
- **Date Utilities**: `year(ts)`, `month(ts)`, `date(ts)`, `day(ts)`, `hour(ts)`, `minute(ts)`, `second(ts)`

## 4. Language Caveats

1. **Mandatory Parentheses**: You cannot reference a function without calling it unless you are intentionally passing it by reference. For zero-argument function invocations, you must use parentheses (e.g., `time()` or `PI()`). 
2. **Semi-Colons**: Statements implicitly return the value of their last expression. Use `;` to sequence operations on the same line if the parser complains, but most newlines will be resolved automatically without them.

## Building and Running

### Compiling

Build the regular tool:
```bash
make
```

Build an optimized version:
```bash
MCA_OPTIMIZE=1 make
```

Build and run tests:
```bash
make bin/test
./bin/test
```

### Usage Help

```bash
USAGE: ./bin/mca <file> [argv]

    -h                  show this help
```
