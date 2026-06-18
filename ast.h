#ifndef AST_H_
#define AST_H_

#include "./lexer.h"
#include "./arena.h"

// @Note: completely arbitrary number. May study what's the best value for this later
#define M_EK_CALL_MAX_ARGUMENTS 32

typedef struct M_Expression M_Expression;

typedef enum {
    M_EK_NUMBER = 0,
    M_EK_ID,
    M_EK_BINARY,
    M_EK_ASSIGN,
    M_EK_UNARY,
    M_EK_EXPRESSION_LIST,
    M_EK_CALL,
    M_EK_LOOP,
    M_EK_BREAK,
    M_EK_IF,
} M_Expression_Kind;

typedef enum {
    M_BINARY_PLUS_OP = 0,
    M_BINARY_TIMES_OP,
    M_BINARY_DIVIDE_OP,
    M_BINARY_SUBTRACT_OP,
    M_BINARY_MOD_OP,
    M_BINARY_POW_OP,

    M_BINARY_EQUAL_OP,
    M_BINARY_NOT_EQUAL_OP,
    M_BINARY_GT_OP,
    M_BINARY_LT_OP,
    M_BINARY_GTE_OP,
    M_BINARY_LTE_OP,
} M_Binary_Expression_Operator;

typedef enum {
    M_UNARY_MINUS_OP,
    M_UNARY_FACTORIAL_OP,
} M_Unary_Expression_Operator;

typedef struct M_Expression_Block M_Expression_Block;

struct M_Expression_Block {
    M_Expression *expr;

    M_Expression_Block *next;
};

struct M_Expression {
    M_Expression_Kind kind;

    union {
        // when the kind is M_EK_NUMBER
        double number;

        // when the kind is M_EK_BREAK it can be null or filled
        M_Expression *expr;

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

        struct {
            const char *fn_name;
            int         fn_name_length;

            M_Expression *arguments[M_EK_CALL_MAX_ARGUMENTS];
            int           arguments_length;
        } call;

        // I'm separating this way because in the future I plan
        // to add metadata to the expressions like location of the token
        // in the file, raw representation, datatype etc, so it will be easier
        // to display error reporting to the user
        struct {
            const char *value;
            int         value_length;
        } id;

        struct {
            struct {
                const char *value;
                int         length;
            } name;

            M_Expression *right;
        } assign;

        struct {
            M_Expression        *condition;
            M_Expression_Block  *block;
        } loop;

        // this is the same as loop for now, but in the future I plan to add else if and else blocks to it,
        // so it will be easier to extend it later
        struct {
            M_Expression        *condition;
            M_Expression_Block  *then_block;
            M_Expression_Block  *else_block;
        } if_expr;
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
