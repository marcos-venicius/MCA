# MCA Language (Math Expression CAlculator)

MCA is an advanced, expression-oriented mathematical calculator and programming language. It is designed to evaluate complex mathematical expressions, with almost everything in the language evaluating to a value.

https://github.com/user-attachments/assets/d7a7ae56-7963-4d62-8066-676eb96ce89b

## 1. Introduction and Syntax Overview

MCA looks similar to C or Rust but acts as an expression evaluator. It features variables, control flow (`if`, `elif`, `else`, `while`), and robust math builtins.

*Note: Semi-colons are not needed in the majority of the cases (the parser will try its best to know where would need one). In some special cases though (where the parser couldn't do its job) you can specify it manually. Most of the times, only `break` will need a special attention.*

### Data Types

MCA has three core data types:
- **Integer**: 64-bit signed integer (`int64_t`).
- **Float**: 64-bit floating point number (`double`).
- **Boolean**: `true` or `false`.

Variables dynamically hold any of these types. When performing arithmetic operations, types are automatically coerced. For example, integer division that results in a remainder will return a float (e.g., `5 / 2` evaluates to `2.5`).

### Operators
- **Arithmetic**: `+`, `-`, `*`, `/`, `%` (modulo), `^` (exponentiation), `!` (factorial)
- **Relational**: `==`, `!=`, `<`, `>`, `<=`, `>=` (these return boolean values)
- **Logical**: `and`, `or` (these return boolean values)

### Control Flow

Because MCA is expression-oriented, blocks and loops evaluate to a value (the last executed expression).

```python
# If-else evaluates to a value
result = if 10 == 10 { 1337 } else { 42 } # result is 1337

# Loops evaluate to a value
x = 0
loop_result = while x < 10 {
    x = x + 1
    if x == 5 { 
        break 42; # Break early and return 42
        # since break accepts an expression as value
        # it's commonly needed to specify a ';' at the end,
        # just to ensure that the next expression on the next line
        # will not be the value of the 'break' if you don't want to.
    }
}
```

## 2. Examples

**Fibonacci Sequence**
```python
target = 15
a = 0
b = 1
n = 0

LAST_FIB_VALUE = while n < target {
  temp = a
  a = b
  b = temp + b
  n = n + 1

  println(a)
}
println(LAST_FIB_VALUE)
```

**Checking Leap Years**
```python
n    = 0
year = year(-3) # Get year from timestamp (using -3 timezone offset)

while n < 15 {
  if (year % 4 == 0 and year % 100 != 0) or (year % 400 == 0) {
    println(year)
  }
  year = year - 1
  n = n + 1
}
```

## 3. Built-in Libraries

MCA provides a wide range of built-in functions. 
*Note: Parentheses are strictly required for function calls, even if they take zero arguments.*

### Constants
- **`PI()`**: Returns $\pi$ ($\approx 3.14159$).
- **`E()`**: Returns Euler's number $e$ ($\approx 2.71828$).

### Math and Rounding
- **`abs(x)`**: Absolute value.
- **`floor(x)`**, **`ceil(x)`**, **`round(x)`**: Rounding functions.
- **`sqrt(x)`**: Square root.
- **`exp(x)`**: Exponential ($e^x$).
- **`log(x)`**, **`log10(x)`**: Natural and base-10 logarithms.
- **`max(x, y, ...)`**, **`min(x, y, ...)`**: Minimum and maximum among arguments.

### Trigonometry (Evaluated in radians)
- **`sin(x)`**, **`cos(x)`**, **`tan(x)`**: Standard trigonometric functions.
- **`rad(x)`**, **`deg(x)`**: Converters between degrees and radians.

### I/O & System
- **`print(...)`**: Prints arguments separated by spaces.
- **`println(...)`**: Prints arguments separated by spaces, followed by a newline.
- **`exit(x)`**: Exits the program with status code `x`.

### Time & Date
- **`time()`**: Returns the current time (timestamp).
- **`year(ts)`**, **`month(ts)`**, **`date(ts)`**, **`day(ts)`**, **`hour(ts)`**, **`minute(ts)`**, **`second(ts)`**: Extracts the specific date/time component from a timestamp `ts`.

### Type Castings and Inspection
- **`type(x)`**: Returns the numeric representation of the type.
- **`as_int(x)`**, **`as_float(x)`**, **`as_bool(x)`**: Explicitly cast `x` to integer, float, or boolean.

## 4. Language Caveats

1. **No String Type**: There are no string literals (like `"hello"`) in the language. Everything revolves around integers, floats, and booleans.
2. **Dynamic Division Coercion**: Division `/` dynamically returns a float or an integer depending on whether there is a fractional remainder. If the result cleanly divides without a fractional part, it evaluates to an `integer` type.
3. **Mandatory Parentheses**: You cannot reference a function without calling it, and you must use parentheses even for zero-argument functions (e.g., `time()` or `PI()`). 
4. **Expression-Oriented Returns**: Statements implicitly return the value of their last expression. Use `;` to sequence operations, but realize that blocks `{ ... }` themselves resolve to a value. 
5. **Trigonometric Inputs**: All trigonometric functions (`sin`, `cos`, `tan`) expect their input in radians, not degrees. Use `rad(degrees)` to wrap and convert your values safely.


## Debugging

you can export `MCA_LOG_ENABLED` as `1` to enable logging.

```bash
export MCA_LOG_ENABLED=1
```

## Building the tool

If you want to build the tool you can use:

```bash
make
```

If you want an omptized version:

```
MCA_OPTIMIZE=1 make
```

## Building the test cases

```bash
make bin/test
```

Running the test cases

```bash
./bin/test
```

## Tool usage help

```bash
USAGE: mca [math] [flags]

    -i   [file]         evaluate math inside a file
    -h                  show this help

error: please, provide some math or -i flag
```
