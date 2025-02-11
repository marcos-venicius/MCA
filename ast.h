#ifndef AST_H_
#define AST_H_

#include "./lexer.h"

typedef struct M_Expression M_Expression;

typedef enum {
    M_EK_NUMBER = 0,
    M_EK_BINARY,
} M_Expression_Kind;

typedef enum {
    M_OP_PLUS = 0,
    M_OP_TIMES,
    M_OP_DIVIDE,
    M_OP_SUBTRACT,
    M_OP_MOD,
    M_OP_POW,
} M_Expression_Operator;

struct M_Expression {
    M_Expression_Kind kind;

    union {
        // when the kind is M_EK_NUMBER
        double number;

        // when the kind is M_EK_BINARY
        struct {
            M_Expression_Operator operator;

            M_Expression *left;
            M_Expression *right;
        } binary;
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
