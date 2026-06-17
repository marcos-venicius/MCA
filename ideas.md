# Future (possible) Feature Ideas for MCA

This document outlines a roadmap of features to implement next to deepen my understanding of compilers and interpreters. 

### 1. Variables and Symbol Tables (State Management)
**Feature:** Allow assigning values to variables and using them in expressions (e.g., `let x = 10; x * 2;`).
**Goals / Concepts to learn:** I will need to build an **Environment** (or Symbol Table). This will teach me how to map string identifiers to values in memory, handle state between expressions, and deal with scope (if I add block statements later). It will also require adding new AST nodes like `M_EK_ASSIGNMENT` and `M_EK_IDENTIFIER`, and extending the lexer to read alphanumeric variable names.

### 2. User-Defined Functions and Closures

**Feature:** Allow defining custom math functions: `def area(r) = 3.14 * r ^ 2; area(5);`
**Goals / Concepts to learn:** I will learn about local scoping. When evaluating `area(5)`, the evaluator needs to push a new "local environment" where `r = 5`, evaluate the AST of the function body, and then pop the environment. This is the first step toward building a Turing-complete language!

### 3. Built-in Functions (Call Expressions)
**Feature:** Add support for math functions like `sin(x)`, `cos(x)`, `max(a, b)`, etc.
**Goals / Concepts to learn:** I will learn how to parse comma-separated argument lists (e.g., `max(1 + 2, 5 * 3)`). This requires a new AST node (`M_EK_CALL`) that holds an identifier and an array of evaluated expressions. In the evaluator, I'll learn how to map identifiers to underlying C-level function pointers.

```python
# we could define functions like:

def area(b, h) = (1/2) * b * h;

# and use the function like this:

area(4, 3) * 2;

# and we could define variables like:

def PI = 3.1415;

# and use as:

PI * 2;
```

So, the unique difference between a function and a variable is that a function would receive one or more arguments.

This gets me to the point where `PI` isn't even a variable, is literally just a function with no arguments.

So, I cannot modify a function, it's immutable, but I could overwrite (or redefine) like:

```python
def PI = 3.1415;

PI * 2;

def PI = 3.14; # this would completely overwrite from now to the next expressions the definition of PI

# So, this would let us do some interesting things:

def PI = PI + 0.15; # replace old PI function with the current value of PI plus 0.15

```

And, once the "variable" is in fact a function, the same would work for them:

```python
def D(x1, y1, x2, y2) = sqrt((x1 - x2) ^ 2 + (y1 - y2) ^ 2);

D(10, 20, 20, 10);

def D(a, b) = abs(a - b);

D(10, 20);
```

This assumes we already is capable of having builtin functions like: `sqrt`, `abs`, `...`.

### 4. Graceful Error Reporting and Synchronization
**Feature:** Instead of calling `exit(1)` upon the first syntax error, collect the error, print a clear error message pointing to the exact line and column (e.g., like Rust or Clang does), and try to "synchronize" the parser to continue finding other errors.
**Goals / Concepts to learn:** I will learn about **Panic Mode Error Recovery**. When an error hits, I will discard tokens until finding a safe synchronization point (like a semicolon `;`), then resume parsing. This is crucial for building real-world compilers that give a complete list of errors instead of crashing at the first typo.

### 5. The Ultimate Challenge: A Bytecode Virtual Machine (VM)
**Feature:** Stop evaluating the AST directly via an `evaluate_expression` tree-walk. Instead, write a compiler pass that translates the AST into a flat array of instructions (Bytecode) like `PUSH 5`, `PUSH 10`, `ADD`, and then write a Stack-Based Virtual Machine to execute those instructions.
**Goals / Concepts to learn:** AST-walking interpreters are inherently slow due to constant pointer dereferencing and recursion overhead. By flattening the tree into bytecode and executing it in a tight `while(true) switch(instruction)` loop, I will learn exactly how Python, Java, and V8 (JavaScript) execute code under the hood at high performance.
