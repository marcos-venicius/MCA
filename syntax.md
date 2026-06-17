# MCA Math Language Syntax Specification

This document outlines the syntax and grammar for the MCA math expression evaluator language.

## Tokens

- **Numbers**: Parsed as floating-point numbers (C `double`).
- **Identifiers**: Alphabetic names used for functions and constants.
- **Operators**: 
  - `+` (Addition)
  - `-` (Subtraction / Unary Minus)
  - `*` (Multiplication)
  - `/` (Division)
  - `%` (Modulo)
  - `^` (Exponentiation / Power)
  - `!` (Factorial)
- **Symbols**:
  - `(` (Left Parenthesis)
  - `)` (Right Parenthesis)
  - `,` (Comma - Argument Separator)
  - `;` (Expression Separator)

## Comments

Single-line comments are supported and are initiated using the `#` character. Any text following a `#` on the same line will be ignored by the parser.


## Operator Precedence and Associativity

Operators are listed from highest to lowest precedence:

| Precedence | Operator(s) | Description                     | Associativity |
|------------|-------------|---------------------------------|---------------|
| 1          | `( ... )`   | Grouping / Parentheses          | N/A           |
| 1          | `fn(...)`   | Function Calls                  | N/A           |
| 2          | `!`         | Factorial (Postfix unary)       | Left          |
| 3          | `-`         | Unary Minus (Prefix unary)      | Right         |
| 4          | `^`         | Exponentiation / Power (Binary) | Right         |
| 5          | `*`, `/`, `%`| Multiplicative (Binary)         | Left          |
| 6          | `+`, `-`    | Additive (Binary)               | Left          |
| 7          | `;`         | Expression Separator            | Left          |

> **Note:** The unary minus (`-`) can only be applied directly once per operand. Successive unary minuses (e.g., `--1`) require grouping via parentheses: `-(-1)`.

## Built-in Functions and Constants

The language supports several built-in functions and constants. Function calls require parentheses, even if they take no arguments.

### Constants
- `pi()`: Returns the value of π (3.14159...)
- `e()`: Returns the value of Euler's number (2.71828...)

### Math Functions
- `abs(x)`: Absolute value
- `max(x, y)`: Maximum of two values
- `min(x, y)`: Minimum of two values
- `sin(x)`: Sine (argument in radians)
- `cos(x)`: Cosine (argument in radians)
- `tan(x)`: Tangent (argument in radians)
- `rad(x)`: Degrees to radians conversion
- `deg(x)`: Radians to degrees conversion
- `sqrt(x)`: Square root
- `log(x)`: Natural logarithm (base e)
- `log10(x)`: Base-10 logarithm
- `exp(x)`: Exponential ($e^x$)
- `floor(x)`: Largest integer not greater than x
- `ceil(x)`: Smallest integer not less than x
- `round(x)`: Nearest integer to x

## Formal Grammar (EBNF-like)

```ebnf
<program>      ::= <expression_list> EOF

<expression_list> ::= <expression> ( ";" <expression> )* [ ";" ]

<expression>   ::= <term> ( ( "+" | "-" ) <term> )*

<term>         ::= <power> ( ( "*" | "/" | "%" ) <power> )*

<power>        ::= <unary> [ "^" <power> ]

<unary>        ::= "-" <factorial> 
                 | <factorial>

<factorial>    ::= <primary> "!"*

<primary>      ::= NUMBER 
                 | IDENTIFIER "(" [ <expression> ( "," <expression> )* ] ")"
                 | "(" <expression> ")"
```

## Evaluation and Separators

Multiple expressions can be provided in the same input by separating them with a semicolon (`;`). 
The language evaluates the input as an expression list. 
If a trailing semicolon is provided at the end of the input (e.g., `1 + 1;`), an empty expression (`<empty>`) is appended to the list and evaluated.
