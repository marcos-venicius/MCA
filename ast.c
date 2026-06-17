#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <assert.h>
#include <string.h>

#include "./ast.h"
#include "./arena.h"

static M_Expression *parse_expression_impl(M_Token **tokens, Clibs_Arena *arena);

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

static M_Expression *parse_primary_expression(M_Token **tokens, Clibs_Arena *arena) {
    if (*tokens == NULL) return NULL;

    M_Token *current = *tokens;

    if (current->kind == M_NUMBER) {
        M_Expression *expr = clibs_arena_alloc(arena, sizeof(M_Expression));
        expr->kind = M_EK_NUMBER;
        expr->number = convert_to_double(current);

        *tokens = current->next;

        return expr;
    } else if (current->kind == M_LPAREN) {
        *tokens = current->next;

        M_Expression *expr = parse_expression_impl(tokens, arena);

        if (*tokens == NULL || (*tokens)->kind != M_RPAREN) {
            fprintf(stderr, "syntax error\n");
            exit(1);
        }

        *tokens = (*tokens)->next;

        return expr;
    } else {
        fprintf(stderr, "unexpected token\n");
        exit(1);
    }

    return NULL;
}

static M_Expression *parse_factorial_expression(M_Token **tokens, Clibs_Arena *arena) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_primary_expression(tokens, arena);

    while (*tokens != NULL && (*tokens)->kind == M_FACTORIAL) {
        M_Token *op_token = *tokens;

        *tokens = (*tokens)->next;

        M_Expression *expr = clibs_arena_alloc(arena, sizeof(M_Expression));

        expr->kind = M_EK_UNARY;
        expr->unary.op = token_kind_as_unary_expression_operator(op_token->kind);
        expr->unary.operand = left;

        left = expr;
    }

    return left;
}

static M_Expression *parse_unary_expression(M_Token **tokens, Clibs_Arena *arena) {
    if (*tokens == NULL) return NULL;

    if ((*tokens)->kind == M_MINUS) {
        M_Unary_Expression_Operator op = token_kind_as_unary_expression_operator((*tokens)->kind);

        *tokens = (*tokens)->next;

        M_Expression *operand = parse_factorial_expression(tokens, arena);

        if (operand == NULL) {
            fprintf(stderr, "syntax error: missing operand for unary '%s'\n", unary_expression_operator_name(op));
            exit(1);
        }

        M_Expression *expr = clibs_arena_alloc(arena, sizeof(M_Expression));

        expr->kind = M_EK_UNARY;
        expr->unary.op = op;
        expr->unary.operand = operand;

        return expr;
    }

    return parse_factorial_expression(tokens, arena);
}

static M_Expression *parse_power_expression(M_Token **tokens, Clibs_Arena *arena) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_unary_expression(tokens, arena);

    while (*tokens != NULL && ((*tokens)->kind == M_POW)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator((*tokens)->kind);

        *tokens = (*tokens)->next;

        M_Expression *right = parse_power_expression(tokens, arena);

        if (right == NULL) {
            fprintf(stderr, "syntax error: missing right operand for '%s'\n", binary_expression_operator_name(op));
            exit(1);
        }

        M_Expression *expr = clibs_arena_alloc(arena, sizeof(M_Expression));

        expr->kind = M_EK_BINARY;
        expr->binary.op = op;
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

static M_Expression *parse_term_expression(M_Token **tokens, Clibs_Arena *arena) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_power_expression(tokens, arena);

    while (*tokens != NULL && ((*tokens)->kind == M_TIMES || (*tokens)->kind == M_DIVIDE || (*tokens)->kind == M_MOD)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator((*tokens)->kind);

        *tokens = (*tokens)->next;

        M_Expression *right = parse_power_expression(tokens, arena);

        if (right == NULL) {
            fprintf(stderr, "syntax error: missing right operand for '%s'\n", binary_expression_operator_name(op));
            exit(1);
        }

        M_Expression *expr = clibs_arena_alloc(arena, sizeof(M_Expression));
        expr->kind = M_EK_BINARY;
        expr->binary.op = op;
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

static M_Expression *parse_expression_impl(M_Token **tokens, Clibs_Arena *arena) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_term_expression(tokens, arena);

    while (*tokens != NULL && ((*tokens)->kind == M_PLUS || (*tokens)->kind == M_MINUS)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator((*tokens)->kind);

        *tokens = (*tokens)->next;

        M_Expression *right = parse_term_expression(tokens, arena);

        if (right == NULL) {
            fprintf(stderr, "syntax error: missing right operand for '%s'\n", binary_expression_operator_name(op));
            exit(1);
        }

        M_Expression *expr = clibs_arena_alloc(arena, sizeof(M_Expression));

        expr->kind = M_EK_BINARY;
        expr->binary.op = op;
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

#define M_AST_MAX_EXPRESSION_ARRAY_SIZE 256

M_Ast *parse_expression(M_Token **tokens) {
    M_Ast *ast = malloc(sizeof(M_Ast));

    ast->expressions_array_length = 0;
    ast->single_expression_arena = clibs_arena_create(sizeof(M_Expression) * 256, sizeof(M_Expression));;
    ast->expressions_array_arena = clibs_arena_create(sizeof(M_Expression*) * M_AST_MAX_EXPRESSION_ARRAY_SIZE, sizeof(M_Expression*));
    ast->expressions_array = (M_Expression**)ast->expressions_array_arena->buffer;

    ast->expressions_array[ast->expressions_array_length++] = parse_expression_impl(tokens, ast->single_expression_arena);

    while (*tokens != NULL && (*tokens)->kind == M_SEMI) {
        *tokens = (*tokens)->next;

        if (ast->expressions_array_length >= M_AST_MAX_EXPRESSION_ARRAY_SIZE) {
            fprintf(stderr, "error: you exceeded the maximum expressions list size of %d\n", M_AST_MAX_EXPRESSION_ARRAY_SIZE);
            ast_free(ast);
            return NULL;
        }

        M_Expression *expr = parse_expression_impl(tokens, ast->single_expression_arena);

        ast->expressions_array[ast->expressions_array_length++] = expr;
    }

    if (*tokens != NULL) {
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
