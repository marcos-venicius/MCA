# MCA Language Syntax Specification

This document provides a highly detailed specification of the syntax, grammar, and standard library for the MCA expression evaluator language.

---

## 1. Tokens and Basic Syntax

- **Numbers**: Parsed as standard floating-point numbers (C `double`). They can contain decimal points (e.g., `3.14`, `10`).
- **Identifiers**: Alphabetic or alphanumeric names starting with a letter or underscore, used for variables, functions, and constants (e.g., `x`, `my_var_1`).
- **Keywords**: Reserved words with specific control flow logic: `while`, `break`, `if`, `else`.
- **Operators**: 
  - `=` (Assignment)
  - `+`, `-`, `*`, `/`, `%`, `^`, `!` (Arithmetic)
  - `==`, `!=`, `<`, `>`, `<=`, `>=` (Relational)
- **Symbols**:
  - `(` `)` (Parentheses for grouping and function calls)
  - `{` `}` (Braces for code blocks)
  - `,` (Argument Separator)
  - `;` (Expression Separator)

### Comments
Single-line comments are supported using the `#` character. Anything following the `#` until the end of the line is ignored by the parser.
```mca
x = 10; # This is a valid comment
```

---

## 2. Operator Precedence and Associativity

Operators evaluate based on the following precedence hierarchy (from highest precedence at 1 to lowest precedence at 10):

| Precedence | Operator(s) | Description                     | Associativity |
|------------|-------------|---------------------------------|---------------|
| 1          | `( ... )`, `{ ... }` | Grouping & Blocks      | N/A           |
| 1          | `fn(...)`   | Function Calls                  | N/A           |
| 1          | `while`, `if`, `break` | Control Flow       | N/A           |
| 2          | `!`         | Factorial (Postfix unary)       | Left          |
| 3          | `-`         | Unary Minus (Prefix unary)      | Right         |
| 4          | `^`         | Exponentiation / Power (Binary) | Right         |
| 5          | `*`, `/`, `%`| Multiplicative (Binary)         | Left          |
| 6          | `+`, `-`    | Additive (Binary)               | Left          |
| 7          | `<`, `<=`, `>`, `>=` | Relational         | Left          |
| 8          | `==`, `!=`  | Equality                        | Left          |
| 9          | `=`         | Assignment                      | Right         |
| 10         | `;`         | Expression Separator            | Left          |

---

## 3. Standard Library (Built-in Functions)

MCA comes with a robust standard library of mathematical functions and constants. Since all values in MCA are `double` floating-point numbers, all functions expect and return `double` numbers. Function calls require parentheses even if they take zero arguments.

## Syscalls

- **`exit(x)`**: Runs C's `exit` function under the hood passing `x` as exit code.

### Constants
These functions return mathematical constants.
- **`pi()`**: Returns the value of $\pi$ ($\approx 3.1415926535$).
- **`e()`**: Returns the value of Euler's number $e$ ($\approx 2.7182818284$).

### Arithmetic and Rounding
- **`abs(x)`**: Returns the absolute (positive) value of `x`.
- **`floor(x)`**: Rounds `x` downwards to the nearest integer.
- **`ceil(x)`**: Rounds `x` upwards to the nearest integer.
- **`round(x)`**: Rounds `x` to the closest integer.
- **`sqrt(x)`**: Returns the square root of `x`.
- **`exp(x)`**: Returns the exponential of `x` ($e^x$).
- **`log(x)`**: Returns the natural logarithm (base $e$) of `x`.
- **`log10(x)`**: Returns the base-10 logarithm of `x`.

### Comparisons
- **`max(x, y)`**: Evaluates and returns the maximum of the two arguments.
- **`min(x, y)`**: Evaluates and returns the minimum of the two arguments.

### Trigonometry
*Note: The trigonometric functions evaluate arguments in radians.*
- **`sin(x)`**: Returns the sine of an angle `x` (in radians).
- **`cos(x)`**: Returns the cosine of an angle `x` (in radians).
- **`tan(x)`**: Returns the tangent of an angle `x` (in radians).
- **`rad(x)`**: Converts degrees to radians. Useful for wrapping arguments: `sin(rad(90))`.
- **`deg(x)`**: Converts radians to degrees.

### I/O (Input/Output)
- **`print(...)`**: Evaluates an arbitrary number of arguments and prints them to standard output separated by spaces. **Returns:** The value of the *last* evaluated argument. 
  *Example:* `print(10, 20, 30)` prints `10.000000 20.000000 30.000000` and evaluates to `30.000000`.

---

## 4. Control Flow and Evaluation

One of MCA's strongest features is its **expression-oriented** nature. Everything evaluates to a value, including blocks `{ ... }`, conditionals, and loops.

### Conditionals (`if` / `else`)
If the condition block evaluates to a non-zero value, the `if` block executes. Otherwise, the `else` block executes. The entire structure evaluates to the last expression executed within the block.
```mca
result = if 10 == 10 { 1337 } else { 42 }; # result is 1337
```

### Loops (`while`)
The `while` loop iterates as long as the condition evaluates to a non-zero value.
If no condition is provided, it acts as an infinite loop. 
```mca
i = 0;
while i < 5 {
    print(i);
    i = i + 1;
};
```

### Early Returns (`break`)
Because loops evaluate to values, you can use `break <expression>` to stop the loop and assign its result immediately.
```mca
x = 0;
result = while {
    x = x + 1;
    if x == 10 { 
        break 42;  # The loop evaluating this break will evaluate to 42
    };
};
# result is 42
```

---

## 5. Formal Grammar (EBNF)

```ebnf
<program>      ::= <expression_list> EOF

<expression_list> ::= <expression> ( ";" <expression> )* [ ";" ]

<expression>   ::= <assignment>

<assignment>   ::= IDENTIFIER "=" <expression>
                 | <equality>

<equality>     ::= <relational> ( ( "==" | "!=" ) <relational> )*

<relational>   ::= <additive> ( ( "<" | "<=" | ">" | ">=" ) <additive> )*

<additive>     ::= <term> ( ( "+" | "-" ) <term> )*

<term>         ::= <power> ( ( "*" | "/" | "%" ) <power> )*

<power>        ::= <unary> [ "^" <power> ]

<unary>        ::= "-" <factorial> 
                 | <factorial>

<factorial>    ::= <primary> "!"*

<primary>      ::= NUMBER 
                 | IDENTIFIER "(" [ <expression> ( "," <expression> )* ] ")"
                 | "while" [ <expression> ] "{" <expression_list> "}"
                 | "if" <expression> "{" <expression_list> "}" [ "else" "{" <expression_list> "}" ]
                 | "break" [ <expression> ]
                 | IDENTIFIER
                 | "(" <expression> ")"
```
