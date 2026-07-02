#pragma once

#include "./lexer.h"
#include "./arena.h"
#include "./location.h"
#include <stdint.h>

// @Leak @Note: completely arbitrary number. May study what's the best value for this later
#define M_EK_CALL_MAX_ARGUMENTS 32

typedef int64_t M_Int;
typedef double  M_Float;
typedef bool    M_Bool;

typedef struct {
    char *value;
    int   value_length;
} M_String;

typedef struct {
    // @Note: just a pointer in a bigger string
    //        cannot be freed
    const char *value;
    int         value_length;
} M_Const_String;

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
    M_EK_ADD_ASSIGN,
    M_EK_SUB_ASSIGN,
    M_EK_UNARY,
    M_EK_FN,
    M_EK_CALL,
    M_EK_WHILE,
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
    M_BINARY_OP_COUNT
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

typedef struct {
    M_Binary_Expression_Operator op;

    M_Expression *left;
    M_Expression *right;
} m_binary_expression_t;

typedef struct {
    M_Unary_Expression_Operator op;

    M_Expression *operand;
} m_unary_expression_t;

typedef struct {
    M_Const_String name;

    // @Note: all the arguments (at least for now) will be an identifier
    //        we do not support default argument values for now
    M_Expression **arguments;
    int            arguments_length;

    M_Expression_Block *block;
} m_fn_expression_t;

typedef struct {
    M_Const_String fn_name;

    // @Leak TODO: improve this to use a dynamic array
    M_Expression *arguments[M_EK_CALL_MAX_ARGUMENTS];
    int           arguments_length;
} m_call_expression_t;

typedef struct {
    M_Const_String name;

    M_Expression *right;
} m_assign_expression_t;

typedef struct {
    M_Expression        *condition;
    M_Expression_Block  *block;
} m_while_loop_expression_t;

typedef struct {
    M_Expression            *condition;
    M_Expression_Block      *then_block;
    M_Expression_Elif_Block *elif_blocks;
    M_Expression_Block      *else_block;
} m_if_expression_t;

struct M_Expression {
    M_Expression_Kind kind;
    M_Location        location;

    union {
        // M_EK_UNIT (void)
        M_Int                     Int;    // M_EK_INT
        M_Float                   Float;  // M_EK_FLOAT
        M_Bool                    Bool;   // M_EK_BOOL
        M_String                  String; // M_EK_STRING @Leak @Note: when it's a string, this will be heap-allocated
        M_Const_String            Id;     // M_EK_ID
        m_binary_expression_t     Binary; // M_EK_BINARY
        m_assign_expression_t     Assign; // M_EK_ASSIGN, M_EK_ADD_ASSIGN, M_EK_SUB_ASSIGN
        m_unary_expression_t      Unary;  // M_EK_UNARY
        m_fn_expression_t         Fn;     // M_EK_FN
        m_call_expression_t       Call;   // M_EK_CALL
        m_while_loop_expression_t While;  // M_EK_WHILE
        m_if_expression_t         If;     // M_EK_IF
        M_Expression             *Break;  // M_EK_BREAK (can be null)
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
    M_Token *last_consumed_token;

    const char *filename;
} M_Ast;

M_Ast *parse_expression(const char *filename, M_Token *head);
const char *binary_expression_operator_name(M_Binary_Expression_Operator op);
void ast_free(M_Ast *ast);