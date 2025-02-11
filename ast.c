#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <assert.h>
#include <string.h>
#include "./ast.h"

static double convert_to_double(M_Token *token) {
    char buffer[token->size + 1];

    memcpy(buffer, token->value, token->size);
    buffer[token->size] = '\0';

    return strtod(buffer, NULL);
}

static M_Expression_Operator from_token_kind_to_expression_operator(M_Token_Kind kind) {
    switch (kind) {
        case M_PLUS:
            return M_OP_PLUS;
        case M_TIMES:
            return M_OP_TIMES;
        case M_MINUS:
            return M_OP_SUBTRACT;
        case M_DIVIDE:
            return M_OP_DIVIDE;
        case M_MOD:
            return M_OP_MOD;
        case M_POW:
            return M_OP_POW;
        default:
            assert(0 && "from_token_kind_to_expression_operator: unreacheable");
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

        M_Expression *expr = parse_expression(tokens);

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

static M_Expression *parse_term(M_Token **tokens) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_primary_expression(tokens);

    while (*tokens != NULL && ((*tokens)->kind == M_TIMES || (*tokens)->kind == M_DIVIDE || (*tokens)->kind == M_MOD || (*tokens)->kind == M_POW)) {
        M_Token *op_token = *tokens;

        *tokens = (*tokens)->next;

        M_Expression *right = parse_primary_expression(tokens);

        M_Expression *expr = malloc(sizeof(M_Expression));
        expr->kind = M_EK_BINARY;
        expr->binary.operator = from_token_kind_to_expression_operator(op_token->kind);
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}

M_Expression *parse_expression(M_Token **tokens) {
    if (*tokens == NULL) return NULL;

    M_Expression *left = parse_term(tokens);

    while (*tokens != NULL && ((*tokens)->kind == M_PLUS || (*tokens)->kind == M_MINUS)) {
        M_Token *op_token = *tokens;

        *tokens = (*tokens)->next;

        M_Expression *right = parse_term(tokens);

        M_Expression *expr = malloc(sizeof(M_Expression));

        expr->kind = M_EK_BINARY;
        expr->binary.operator = from_token_kind_to_expression_operator(op_token->kind);
        expr->binary.left = left;
        expr->binary.right = right;

        left = expr;
    }

    return left;
}
