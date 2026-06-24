# Future (possible) Feature Ideas for MCA

This document outlines a roadmap of features to implement next to deepen my understanding of compilers and interpreters. 

~### 1. Variables and Symbol Tables (State Management)~
~**Feature:** Allow assigning values to variables and using them in expressions (e.g., `let x = 10; x * 2;`).~
~**Goals / Concepts to learn:** I will need to build an **Environment** (or Symbol Table). This will teach me how to map string identifiers to values in memory, handle state between expressions, and deal with scope (if I add block statements later). It will also require adding new AST nodes like `M_EK_ASSIGNMENT` and `M_EK_IDENTIFIER`, and extending the lexer to read alphanumeric variable names.~

### 2. User-Defined Functions and Closures

**In progress**

**Feature:** Allow defining custom math functions: `def area(r) = 3.14 * r ^ 2; area(5);`
**Goals / Concepts to learn:** I will learn about local scoping. When evaluating `area(5)`, the evaluator needs to push a new "local environment" where `r = 5`, evaluate the AST of the function body, and then pop the environment. This is the first step toward building a Turing-complete language!

We are going to have some limitations for now.

That is going to be the syntax of a function definition:

```r
# inline definitions
def fib(n, a, b) = if n <= 0 { a } else { fib(n - 1, b, a + b) };

println(fib(10, 0, 1));

# multi-line definitions
def fib(n) = {
    a = 0
    b = 1

    while n > 0 {
        t = a
        a = b
        b = t + b

        n = n - 1
    }

    a
}

println(fib(10));
```

~### 3. Built-in Functions (Call Expressions)~
~**Feature:** Add support for math functions like `sin(x)`, `cos(x)`, `max(a, b)`, etc.~
~**Goals / Concepts to learn:** I will learn how to parse comma-separated argument lists (e.g., `max(1 + 2, 5 * 3)`). This requires a new AST node (`M_EK_CALL`) that holds an identifier and an array of evaluated expressions. In the evaluator, I'll learn how to map identifiers to underlying C-level function pointers.~

~### 4. Graceful Error Reporting and Synchronization~
~**Feature:** Instead of calling `exit(1)` upon the first syntax error, collect the error, print a clear error message pointing to the exact line and column (e.g., like Rust or Clang does), and try to "synchronize" the parser to continue finding other errors.~
~**Goals / Concepts to learn:** I will learn about **Panic Mode Error Recovery**. When an error hits, I will discard tokens until finding a safe synchronization point (like a semicolon `;`), then resume parsing. This is crucial for building real-world compilers that give a complete list of errors instead of crashing at the first typo.~

### 5. The Ultimate Challenge: A Bytecode Virtual Machine (VM)

> [!NOTE]
> it's not planned for now

**Feature:** Stop evaluating the AST directly via an `evaluate_expression` tree-walk. Instead, write a compiler pass that translates the AST into a flat array of instructions (Bytecode) like `PUSH 5`, `PUSH 10`, `ADD`, and then write a Stack-Based Virtual Machine to execute those instructions.
**Goals / Concepts to learn:** AST-walking interpreters are inherently slow due to constant pointer dereferencing and recursion overhead. By flattening the tree into bytecode and executing it in a tight `while(true) switch(instruction)` loop, I will learn exactly how Python, Java, and V8 (JavaScript) execute code under the hood at high performance.
