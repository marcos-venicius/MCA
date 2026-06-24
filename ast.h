#ifndef AST_H_
#define AST_H_

#include "./lexer.h"
#include "./arena.h"
#include "./location.h"
#include <stdint.h>

// @Note: completely arbitrary number. May study what's the best value for this later
#define M_EK_CALL_MAX_ARGUMENTS 32

typedef struct {
    char *value;
    int   value_length;
} M_String;

typedef struct M_Expression M_Expression;

typedef enum {
    M_EK_UNIT = 0,
    M_EK_INT,
    M_EK_FLOAT,
    M_EK_BOOL,
    M_EK_STRING,
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

    M_BINARY_AND_OP,
    M_BINARY_OR_OP,

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
    M_UNARY_NOT_OP,
} M_Unary_Expression_Operator;

typedef struct M_Expression_Block M_Expression_Block;

struct M_Expression_Block {
    M_Expression *expr;

    M_Expression_Block *next;
};

typedef struct M_Expression_Elif_Block M_Expression_Elif_Block;

struct M_Expression_Elif_Block {
    M_Expression        *condition;
    M_Expression_Block  *block;

    M_Expression_Elif_Block *next;
};

struct M_Expression {
    M_Expression_Kind kind;
    M_Location location;

    union {
        int64_t integer;
        double  floating;
        bool    boolean;

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

        // @Note: when it's a string, this will be heap-allocated
        // when it's and ID it'll be a sized string using just
        // a pointer to the original string during lexing
        // TODO: should I own everything?
        struct {
            const char *value;
            int         value_length;
        } id;
        M_String string;

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

        struct {
            M_Expression            *condition;
            M_Expression_Block      *then_block;
            M_Expression_Elif_Block *elif_blocks;
            M_Expression_Block      *else_block;
        } if_expr;
    };
};

typedef struct {
    Clibs_Arena  *single_expression_arena;

    Clibs_Arena  *block_expression_arena;
    Clibs_Arena   *expressions_array_arena;
    M_Expression **expressions_array;
    int            expressions_array_length;

    size_t errors;

    M_Token *current_token;

    const char *filename;
} M_Ast;

M_Ast *parse_expression(const char *filename, M_Token *head);
const char *binary_expression_operator_name(M_Binary_Expression_Operator op);
void ast_free(M_Ast *ast);

#endif // AST_H_
