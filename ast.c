#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <assert.h>
#include <string.h>
#include "./ast.h"

// TODO: use arena to the expressions

static M_Expression *parse_expression_impl(M_Token **tokens);

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

static M_Expression *parse_primary_expression(M_Token **tokens) {
    if (*tokens == NULL) return NULL;

    M_Token *current = *tokens;

    if (current->kind == M_NUMBER) {
        M_Expression *expr = malloc(sizeof(M_Expression));
        expr->kind = M_EK_NUMBER;
        expr->number = convert_to_double(current);

        *tokens = current->next;

        return expr;
    } else if (current->kind == M_LPAREN) {
        *tokens = current->next;

        M_Expression *expr = parse_expression_impl(tokens);

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

static M_Expression *parse_factorial_expression(M_Token **tokens) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_primary_expression(tokens);

    while (*tokens != NULL && (*tokens)->kind == M_FACTORIAL) {
        M_Token *op_token = *tokens;

        *tokens = (*tokens)->next;

        M_Expression *expr = malloc(sizeof(M_Expression));

        expr->kind = M_EK_UNARY;
        expr->unary.op = token_kind_as_unary_expression_operator(op_token->kind);
        expr->unary.operand = left;

        left = expr;
    }

    return left;
}

static M_Expression *parse_unary_expression(M_Token **tokens) {
    if (*tokens == NULL) return NULL;

    if ((*tokens)->kind == M_MINUS) {
        M_Token *op_token = *tokens;

        *tokens = (*tokens)->next;

        M_Expression *expr = malloc(sizeof(M_Expression));

        expr->kind = M_EK_UNARY;
        expr->unary.op = token_kind_as_unary_expression_operator(op_token->kind);
        expr->unary.operand = parse_factorial_expression(tokens);

        return expr;
    }

    return parse_factorial_expression(tokens);
}

static M_Expression *parse_power_expression(M_Token **tokens) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_unary_expression(tokens);

    while (*tokens != NULL && ((*tokens)->kind == M_POW)) {
        M_Token *op_token = *tokens;

        *tokens = (*tokens)->next;

        M_Expression *right = parse_power_expression(tokens);

        M_Expression *expr = malloc(sizeof(M_Expression));
        expr->kind = M_EK_BINARY;
        expr->binary.op = token_kind_as_binary_expression_operator(op_token->kind);
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

static M_Expression *parse_term_expression(M_Token **tokens) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_power_expression(tokens);

    while (*tokens != NULL && ((*tokens)->kind == M_TIMES || (*tokens)->kind == M_DIVIDE || (*tokens)->kind == M_MOD)) {
        M_Token *op_token = *tokens;

        *tokens = (*tokens)->next;

        M_Expression *right = parse_power_expression(tokens);

        M_Expression *expr = malloc(sizeof(M_Expression));
        expr->kind = M_EK_BINARY;
        expr->binary.op = token_kind_as_binary_expression_operator(op_token->kind);
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

static M_Expression *parse_expression_impl(M_Token **tokens) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_term_expression(tokens);

    while (*tokens != NULL && ((*tokens)->kind == M_PLUS || (*tokens)->kind == M_MINUS)) {
        M_Token *op_token = *tokens;

        *tokens = (*tokens)->next;

        M_Expression *right = parse_term_expression(tokens);

        M_Expression *expr = malloc(sizeof(M_Expression));

        expr->kind = M_EK_BINARY;
        expr->binary.op = token_kind_as_binary_expression_operator(op_token->kind);
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

M_Expression *parse_expression(M_Token **tokens) {
    M_Expression *left = parse_expression_impl(tokens);

    while (*tokens != NULL && (*tokens)->kind == M_SEMI) {
        *tokens = (*tokens)->next;

        if (left->kind == M_EK_EXPRESSION_LIST) {
            left->expressions_list.expressions = realloc(
                left->expressions_list.expressions,
                sizeof(M_Expression*) * (left->expressions_list.expressions_length + 1)
            );

            M_Expression *expr = parse_expression_impl(tokens);

            left->expressions_list.expressions[left->expressions_list.expressions_length++] = expr;
        } else {
            M_Expression *expr = malloc(sizeof(M_Expression));

            expr->kind = M_EK_EXPRESSION_LIST;
            expr->expressions_list.expressions = malloc(sizeof(M_Expression*) * 2);
            expr->expressions_list.expressions_length = 2;
            expr->expressions_list.expressions[0] = left;
            expr->expressions_list.expressions[1] = parse_expression_impl(tokens);

            left = expr;
        }
    }

    if (*tokens != NULL) {
        fprintf(stderr, "syntax error\n");
        exit(1);
    }

    return left;
}
