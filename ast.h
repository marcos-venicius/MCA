#ifndef AST_H_
#define AST_H_

#include "./lexer.h"
#include "./arena.h"

typedef struct M_Expression M_Expression;

typedef enum {
    M_EK_NUMBER = 0,
    M_EK_BINARY,
    M_EK_UNARY,
    M_EK_EXPRESSION_LIST,
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

typedef struct {
    Clibs_Arena  *single_expression_arena;

    Clibs_Arena   *expressions_array_arena;
    M_Expression **expressions_array;
    int            expressions_array_length;

    size_t errors;

    M_Token *current_token;

    const char *filename;
} M_Ast;

M_Ast *parse_expression(const char *filename, M_Token *head);
void ast_free(M_Ast *ast);

#endif // AST_H_
