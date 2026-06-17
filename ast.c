#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <assert.h>
#include <string.h>

#include "./ast.h"
#include "./arena.h"

static M_Expression *parse_expression_impl(M_Ast *ast);

static inline M_Token *token(M_Ast *ast) {
    return ast->current_token;
}

static inline void next_token(M_Ast *ast) {
    if (ast->current_token != NULL)
        ast->current_token = ast->current_token->next;
}

static double convert_to_double(M_Token *token) {
    char buffer[token->size + 1];

    memcpy(buffer, token->value, token->size);
    buffer[token->size] = '\0';

    return strtod(buffer, NULL);
}

static M_Binary_Expression_Operator token_kind_as_binary_expression_operator(M_Token_Kind kind) {
    switch (kind) {
        case M_PLUS:
            return M_BINARY_PLUS_OP;
        case M_TIMES:
            return M_BINARY_TIMES_OP;
        case M_MINUS:
            return M_BINARY_SUBTRACT_OP;
        case M_DIVIDE:
            return M_BINARY_DIVIDE_OP;
        case M_MOD:
            return M_BINARY_MOD_OP;
        case M_POW:
            return M_BINARY_POW_OP;
        default:
            assert(0 && "token_kind_as_binary_expression_operator: unreacheable");
    }
}

static const char *binary_expression_operator_name(M_Binary_Expression_Operator op) {
    switch (op) {
        case M_BINARY_PLUS_OP: return "+";
        case M_BINARY_TIMES_OP: return "*";
        case M_BINARY_SUBTRACT_OP: return "-";
        case M_BINARY_DIVIDE_OP: return "/";
        case M_BINARY_MOD_OP: return "%";
        case M_BINARY_POW_OP: return "^";
        default:
            assert(0 && "binary_expression_operator_name: unreacheable");
    }
}

static const char *unary_expression_operator_name(M_Unary_Expression_Operator op) {
    switch (op) {
        case M_UNARY_MINUS_OP: return "-";
        case M_UNARY_FACTORIAL_OP: return "!";
        default:
            assert(0 && "unary_expression_operator_name: unreacheable");
    }
}

static M_Unary_Expression_Operator token_kind_as_unary_expression_operator(M_Token_Kind kind) {
    switch (kind) {
        case M_MINUS:
            return M_UNARY_MINUS_OP;
        case M_FACTORIAL:
            return M_UNARY_FACTORIAL_OP;
        default:
            assert(0 && "token_kind_as_unary_expression_operator: unreacheable");
    }
}

static M_Expression *parse_primary_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    if (token(ast)->kind == M_NUMBER) {
        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
        expr->kind = M_EK_NUMBER;
        expr->number = convert_to_double(token(ast));

        next_token(ast);

        return expr;
    } else if (token(ast)->kind == M_LPAREN) {
        next_token(ast);

        M_Expression *expr = parse_expression_impl(ast);

        if (token(ast) == NULL || token(ast)->kind != M_RPAREN) {
            fprintf(stderr, "syntax error\n");
            exit(1);
        }

        next_token(ast);

        return expr;
    } else {
        fprintf(stderr, "unexpected token\n");
        exit(1);
    }

    return NULL;
}

static M_Expression *parse_factorial_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_primary_expression(ast);

    while (token(ast) != NULL && token(ast)->kind == M_FACTORIAL) {
        M_Unary_Expression_Operator op = token_kind_as_unary_expression_operator(token(ast)->kind);

        next_token(ast);

        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

        expr->kind = M_EK_UNARY;
        expr->unary.op = op;
        expr->unary.operand = left;

        left = expr;
    }

    return left;
}

static M_Expression *parse_unary_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    if (token(ast)->kind == M_MINUS) {
        M_Unary_Expression_Operator op = token_kind_as_unary_expression_operator(token(ast)->kind);

        next_token(ast);

        M_Expression *operand = parse_factorial_expression(ast);

        if (operand == NULL) {
            fprintf(stderr, "syntax error: missing operand for unary '%s'\n", unary_expression_operator_name(op));
            exit(1);
        }

        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

        expr->kind = M_EK_UNARY;
        expr->unary.op = op;
        expr->unary.operand = operand;

        return expr;
    }

    return parse_factorial_expression(ast);
}

static M_Expression *parse_power_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_unary_expression(ast);

    while (token(ast) != NULL && (token(ast)->kind == M_POW)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator(token(ast)->kind);

        next_token(ast);

        M_Expression *right = parse_power_expression(ast);

        if (right == NULL) {
            fprintf(stderr, "syntax error: missing right operand for '%s'\n", binary_expression_operator_name(op));
            exit(1);
        }

        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

        expr->kind = M_EK_BINARY;
        expr->binary.op = op;
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

static M_Expression *parse_term_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_power_expression(ast);

    while (token(ast) != NULL && (token(ast)->kind == M_TIMES || token(ast)->kind == M_DIVIDE || token(ast)->kind == M_MOD)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator(token(ast)->kind);

        next_token(ast);

        M_Expression *right = parse_power_expression(ast);

        if (right == NULL) {
            fprintf(stderr, "syntax error: missing right operand for '%s'\n", binary_expression_operator_name(op));
            exit(1);
        }

        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
        expr->kind = M_EK_BINARY;
        expr->binary.op = op;
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

static M_Expression *parse_expression_impl(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_term_expression(ast);

    while (token(ast) != NULL && (token(ast)->kind == M_PLUS || token(ast)->kind == M_MINUS)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator(token(ast)->kind);

        next_token(ast);

        M_Expression *right = parse_term_expression(ast);

        if (right == NULL) {
            fprintf(stderr, "syntax error: missing right operand for '%s'\n", binary_expression_operator_name(op));
            exit(1);
        }

        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

        expr->kind = M_EK_BINARY;
        expr->binary.op = op;
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

#define M_AST_MAX_EXPRESSION_ARRAY_SIZE 256

M_Ast *parse_expression(M_Token *head) {
    M_Ast *ast = malloc(sizeof(M_Ast));

    ast->current_token = head;
    ast->expressions_array_length = 0;
    ast->single_expression_arena = clibs_arena_create(sizeof(M_Expression) * 256, sizeof(M_Expression));;
    ast->expressions_array_arena = clibs_arena_create(sizeof(M_Expression*) * M_AST_MAX_EXPRESSION_ARRAY_SIZE, sizeof(M_Expression*));
    ast->expressions_array = (M_Expression**)ast->expressions_array_arena->buffer;

    ast->expressions_array[ast->expressions_array_length++] = parse_expression_impl(ast);

    while (token(ast) != NULL && token(ast)->kind == M_SEMI) {
        next_token(ast);

        if (ast->expressions_array_length >= M_AST_MAX_EXPRESSION_ARRAY_SIZE) {
            fprintf(stderr, "error: you exceeded the maximum expressions list size of %d\n", M_AST_MAX_EXPRESSION_ARRAY_SIZE);
            ast_free(ast);
            return NULL;
        }

        M_Expression *expr = parse_expression_impl(ast);

        ast->expressions_array[ast->expressions_array_length++] = expr;
    }

    if (token(ast) != NULL) {
        fprintf(stderr, "syntax error\n");
        exit(1);
    }

    return ast;
}

void ast_free(M_Ast *ast) {
    clibs_arena_destroy(ast->single_expression_arena);
    clibs_arena_destroy(ast->expressions_array_arena);

    free(ast);
}
