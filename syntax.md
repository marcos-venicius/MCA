# MCA Math Language Syntax Specification

This document outlines the syntax and grammar for the MCA math expression evaluator language.

## Tokens

- **Numbers**: Parsed as floating-point numbers (C `double`).
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
  - `;` (Expression Separator)

## Comments

Single-line comments are supported and are initiated using the `#` character. Any text following a `#` on the same line will be ignored by the parser.


## Operator Precedence and Associativity

Operators are listed from highest to lowest precedence:

| Precedence | Operator(s) | Description                     | Associativity |
|------------|-------------|---------------------------------|---------------|
| 1          | `( ... )`   | Grouping / Parentheses          | N/A           |
| 2          | `!`         | Factorial (Postfix unary)       | Left          |
| 3          | `-`         | Unary Minus (Prefix unary)      | Right         |
| 4          | `^`         | Exponentiation / Power (Binary) | Right         |
| 5          | `*`, `/`, `%`| Multiplicative (Binary)         | Left          |
| 6          | `+`, `-`    | Additive (Binary)               | Left          |
| 7          | `;`         | Expression Separator            | Left          |

> **Note:** The unary minus (`-`) can only be applied directly once per operand. Successive unary minuses (e.g., `--1`) require grouping via parentheses: `-(-1)`.

## Formal Grammar (EBNF-like)

```ebnf
<program>      ::= <expression> EOF

<expression>   ::= <term> ( ( "+" | "-" | ";" ) <term> )*

<term>         ::= <power> ( ( "*" | "/" | "%" ) <power> )*

<power>        ::= <unary> [ "^" <power> ]

<unary>        ::= "-" <factorial> 
                 | <factorial>

<factorial>    ::= <primary> "!"*

<primary>      ::= NUMBER 
                 | "(" <expression> ")"
```

## Evaluation and Separators

Multiple expressions can be provided in the same input by separating them with a semicolon (`;`). 
The language evaluates the input as an expression list. 
If a trailing semicolon is provided at the end of the input (e.g., `1 + 1;`), an empty expression (`<empty>`) is appended to the list and evaluated.
