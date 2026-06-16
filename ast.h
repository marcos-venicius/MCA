#ifndef AST_H_
#define AST_H_

#include "./lexer.h"

typedef struct M_Expression M_Expression;

typedef enum {
    M_EK_NUMBER = 0,
    M_EK_BINARY,
    M_EK_UNARY,
} M_Expression_Kind;

typedef enum {
    M_BINARY_PLUS_OP = 0,
    M_BINARY_TIMES_OP,
    M_BINARY_DIVIDE_OP,
    M_BINARY_SUBTRACT_OP,
    M_BINARY_MOD_OP,
    M_BINARY_POW_OP,
} M_Binary_Expression_Operator;

typedef enum {
    M_UNARY_MINUS_OP,
    M_UNARY_FACTORIAL_OP,
} M_Unary_Expression_Operator;

struct M_Expression {
    M_Expression_Kind kind;

    union {
        // when the kind is M_EK_NUMBER
        double number;

        // when the kind is M_EK_BINARY
        struct {
            M_Binary_Expression_Operator op;

            M_Expression *left;
            M_Expression *right;
        } binary;

        struct {
            M_Unary_Expression_Operator op;

            M_Expression *operand;
        } unary;
    };
};

M_Expression *parse_expression(M_Token **tokens);

#endif // AST_H_
    


/*
 
3 + 4 * 2 = 11

  +
 / \
3   *
   / \
  4   2

(3 + 4) * 2 = 14

    *
   / \
  +   2
 / \
3   4

*/
