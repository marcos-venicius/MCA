# MCA Parser and Evaluator Issues

This document tracks critical logical bugs and edge cases found in the AST parsing and evaluation logic (`ast.c` and `main.c`).

### 1. `M_EK_EXPRESSION_LIST` Union Corruption Inside Parentheses
The `parse_primary_expression` function calls `parse_expression_impl(tokens, false)` to process anything within parentheses. Because `parse_expression_impl` parses semicolons (`;`) into `M_EK_EXPRESSION_LIST` nodes unconditionally, it is possible to syntactically nest an expression list inside a binary expression, such as `(1 ; 2) + 3`. 
The problem arises in `evaluate_expression()`: it explicitly handles `M_EK_NUMBER` and `M_EK_UNARY`, but **assumes any other node is an `M_EK_BINARY` node**. If it tries to evaluate a nested list node, it accesses the `binary.op` and `binary.left` fields, which overlap in the C `union` with the `expressions**` pointer. This results in the evaluator interpreting raw memory addresses as enum integers, leading to severe memory corruption and segfaults.

### 2. Missing Right-Operands Cause Silent Null-Dereference Segfaults
If a user writes an incomplete binary expression like `1 + ` or `5 * ;`, the parser does not throw a syntax error.
Functions like `parse_term_expression` will attempt to parse the right-hand side, but upon hitting `EOF` (or an unexpected operator like `;`), they immediately return `NULL`. The parent parser then blindly creates a binary `M_Expression` node with `right = NULL` without verifying the result. When `evaluate_expression` eventually runs, it attempts to access `expression->binary.right->kind`, dereferencing the `NULL` pointer and causing a segmentation fault.

### 3. Uninitialized Pointer on Empty Input
If an empty file (or a file consisting entirely of comments/whitespace) is passed to `compile_math()`, the lexer produces `tokens == NULL`. The function correctly logs "There is no tokens" and returns `0` (success). 
However, it **never initializes** the `*expression_output` pointer before returning. In `main.c`, the program checks if `result == 0` and immediately reads `expression->kind`. Because `expression` was never initialized, this invokes Undefined Behavior (typically an immediate segfault).

### 4. AST Memory Leak
While there is an `m_lexer_free` function that correctly cleans up the tokens array, there is no corresponding teardown function for the `M_Expression` tree. Every time `parse_expression` completes, all the dynamically allocated AST nodes (and the `expressions_list` dynamic arrays) are completely leaked. While the OS reclaims this memory on exit, if this parser was ever used as a long-living library or evaluated in a loop, it would cause a huge memory leak.
