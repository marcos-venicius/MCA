#include <stdarg.h>
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

static inline bool checkahead(M_Ast *ast, M_Token_Kind kind) {
    return ast->current_token != NULL && ast->current_token->next != NULL && ast->current_token->next->kind == kind;
}

static inline M_Token *next_token(M_Ast *ast) {
    if (ast->current_token != NULL) {
        ast->current_token = ast->current_token->next;

        return ast->current_token;
    }

    return NULL;
}

static double convert_to_double(M_Token *token) {
    char buffer[token->size + 1];

    memcpy(buffer, token->value, token->size);
    buffer[token->size] = '\0';

    return strtod(buffer, NULL);
}

static M_Binary_Expression_Operator token_kind_as_binary_expression_operator(M_Token_Kind kind) {
    switch (kind) {
        case M_PLUS:    return M_BINARY_PLUS_OP;
        case M_TIMES:   return M_BINARY_TIMES_OP;
        case M_MINUS:   return M_BINARY_SUBTRACT_OP;
        case M_DIVIDE:  return M_BINARY_DIVIDE_OP;
        case M_MOD:     return M_BINARY_MOD_OP;
        case M_POW:     return M_BINARY_POW_OP;

        case M_EQUAL:       return M_BINARY_EQUAL_OP;
        case M_NOT_EQUAL:   return M_BINARY_NOT_EQUAL_OP;
        case M_GT:          return M_BINARY_GT_OP;
        case M_LT:          return M_BINARY_LT_OP;
        case M_GTE:         return M_BINARY_GTE_OP;
        case M_LTE:         return M_BINARY_LTE_OP;

        case M_ASSIGN:
        case M_ID:
        case M_NUMBER:
        case M_FACTORIAL:
        case M_LPAREN:
        case M_RPAREN:
        case M_LCURLY:
        case M_RCURLY:
        case M_SEMI:
        case M_COMMA:
            assert(0 && "token_kind_as_binary_expression_operator: invalid token kind as binary operator");
            break;
    }

    assert(0 && "token_kind_as_binary_expression_operator: unreacheable");
}

static const char *binary_expression_operator_name(M_Binary_Expression_Operator op) {
    switch (op) {
        case M_BINARY_PLUS_OP:      return "+";
        case M_BINARY_TIMES_OP:     return "*";
        case M_BINARY_SUBTRACT_OP:  return "-";
        case M_BINARY_DIVIDE_OP:    return "/";
        case M_BINARY_MOD_OP:       return "%";
        case M_BINARY_POW_OP:       return "^";

        case M_BINARY_EQUAL_OP:     return "==";
        case M_BINARY_NOT_EQUAL_OP: return "!=";
        case M_BINARY_GT_OP:        return ">";
        case M_BINARY_LT_OP:        return "<";
        case M_BINARY_GTE_OP:       return ">=";
        case M_BINARY_LTE_OP:       return "<=";
    }

    assert(0 && "binary_expression_operator_name: unreacheable");
}

static const char *unary_expression_operator_name(M_Unary_Expression_Operator op) {
    switch (op) {
        case M_UNARY_MINUS_OP:      return "-";
        case M_UNARY_FACTORIAL_OP:  return "!";
    }

    assert(0 && "unary_expression_operator_name: unreacheable");
}

static M_Unary_Expression_Operator token_kind_as_unary_expression_operator(M_Token_Kind kind) {
    switch (kind) {
        case M_MINUS:       return M_UNARY_MINUS_OP;
        case M_FACTORIAL:   return M_UNARY_FACTORIAL_OP;

        case M_ASSIGN:
        case M_PLUS:
        case M_TIMES:
        case M_DIVIDE:
        case M_MOD:
        case M_POW:
        case M_EQUAL:
        case M_NOT_EQUAL:
        case M_GT:
        case M_LT:
        case M_GTE:
        case M_LTE:
        case M_ID:
        case M_NUMBER:
        case M_LPAREN:
        case M_RPAREN:
        case M_LCURLY:
        case M_RCURLY:
        case M_SEMI:
        case M_COMMA:
            assert(0 && "token_kind_as_unary_expression_operator: invalid token kind as unary operator");
    }

    assert(0 && "token_kind_as_unary_expression_operator: unreacheable");
}

static void synchronize(M_Ast *ast) {
    while (token(ast) != NULL && token(ast)->kind != M_SEMI) next_token(ast);
}

static void ast_error(M_Ast *ast, M_Token *token, const char *message, ...) {
    va_list args;
    va_start(args, message);

    if (ast->filename != NULL) {
        fprintf(stderr, "%s:%d:%d \033[0;31msyntax error\033[0m: ", ast->filename, token->loc.line, token->loc.col);
    } else {
        fprintf(stderr, "%d:%d \033[0;31msyntax error\033[0m: ", token->loc.line, token->loc.col);
    }

    vfprintf(stderr, message, args);
    fprintf(stderr, "\n");

    va_end(args);

    ast->errors++;
}


static M_Expression *parse_function_call_expression(M_Ast *ast) {
    M_Token *fn_name = token(ast);

    M_Token *lparen = next_token(ast);

    if (lparen == NULL) {
        ast_error(ast, fn_name, "expected '(' but got EOF");
        synchronize(ast);
        return NULL;
    }

    M_Token *next = next_token(ast);

    if (next == NULL) {
        ast_error(ast, lparen, "expected ')' or an expression but got EOF");
        synchronize(ast);
        return NULL;
    }

    M_Expression *fn = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

    fn->kind = M_EK_CALL;
    fn->call.fn_name = fn_name->value;
    fn->call.fn_name_length = fn_name->size;
    fn->call.arguments_length = 0;

    // 'identifier()' a function call with no args.
    if (next->kind == M_RPAREN) {
        next_token(ast); // skip ')'

        return fn;
    }

    // consume arguments
    while (token(ast) != NULL && token(ast)->kind != M_RPAREN) {
        M_Expression *expr = parse_expression_impl(ast);

        // something went wrong while parsing expression
        // so we just propagate the error
        if (expr == NULL) return NULL;

        fn->call.arguments[fn->call.arguments_length++] = expr;

        if (token(ast)->kind == M_RPAREN) break;

        if (token(ast)->kind != M_COMMA) {
            ast_error(ast, lparen, "expected ',' but got '%.*s'", token(ast)->size, token(ast)->value);
            synchronize(ast);
            return NULL;
        }

        next_token(ast); // skip ','
    }

    next = token(ast);

    if (next == NULL) {
        ast_error(ast, lparen, "expected ')' but got EOF");
        synchronize(ast);
        return NULL;
    }

    if (next->kind != M_RPAREN) {
        ast_error(ast, lparen, "expected ')' but got '%.*s'", next->size, next->value);
        synchronize(ast);
        return NULL;
    }

    next_token(ast);

    return fn;
}

static M_Expression *parse_variable_expression(M_Ast *ast) {
    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->kind = M_EK_ID;
    expr->id.value = token(ast)->value;
    expr->id.value_length = token(ast)->size;

    next_token(ast);

    return expr;
}

static M_Expression *parse_break_expression(M_Ast *ast) {
    next_token(ast); // jump 'break'

    M_Expression *break_value = NULL;

    if (token(ast) != NULL && token(ast)->kind != M_SEMI)
        break_value = parse_expression_impl(ast);

    M_Expression *loop_break = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

    loop_break->kind = M_EK_BREAK;
    loop_break->expr = break_value;

    return loop_break;
}

static M_Expression *parse_loop_expression(M_Ast *ast) {
    M_Token *first_token = token(ast);

    next_token(ast); // jump keyword

    if (token(ast) == NULL) {
        ast_error(ast, first_token, "unterminated loop expression. expected an expression or a '{' but got EOF");
        synchronize(ast);
        return NULL;
    }

    M_Expression *condition = NULL;

    if (token(ast)->kind != M_LCURLY) {
        condition = parse_expression_impl(ast);

        // just propagating upper errors
        if (condition == NULL) return NULL;
    }

    if (token(ast) == NULL) {
        ast_error(ast, first_token, "unterminated loop expression. expected '{' but got EOF");
        synchronize(ast);
        return NULL;
    }

    if (token(ast)->kind != M_LCURLY) {
        ast_error(ast, first_token, "unterminated loop expression. expected '{' but got '%.*s'", token(ast)->size, token(ast)->value);
        synchronize(ast);
        return NULL;
    }

    next_token(ast); // skip '{'

    // @Leak TODO: maybe, use an arena. Just fix this leak. This block struct is never gonna be 'freed
    M_Expression_Block *block_head = malloc(sizeof(M_Expression_Block));
    block_head->next = NULL;
    block_head->expr = NULL;
    M_Expression_Block *block_tail = NULL;

    // Parse block
    while (token(ast) != NULL && token(ast)->kind != M_RCURLY) {
        if (token(ast) != NULL && token(ast)->kind == M_SEMI) {
            next_token(ast);
            continue;
        }

        M_Expression *expr = parse_expression_impl(ast);

        // just propagating upper errors
        if (expr == NULL) return NULL;

        if (block_tail == NULL) {
            block_head->expr = expr;
            block_head->next = NULL;
            block_tail = block_head;
        } else {
            M_Expression_Block *inner_block = malloc(sizeof(M_Expression_Block));

            inner_block->expr = expr;
            inner_block->next = NULL;

            block_tail->next = inner_block;
            block_tail = block_tail->next;
        }
    }

    if (token(ast) == NULL) {
        ast_error(ast, first_token, "unterminated loop expression. expected '}' but got EOF");
        synchronize(ast);
        return NULL;
    }

    if (token(ast)->kind != M_RCURLY) {
        ast_error(ast, first_token, "unterminated loop expression. expected '}' but got '%.*s", token(ast)->size, token(ast)->value);
        synchronize(ast);
        return NULL;
    }

    next_token(ast); // skip '}'

    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->kind = M_EK_LOOP;
    expr->loop.condition = condition; // means infinite
    expr->loop.block = block_head;

    return expr;
}

static M_Expression *parse_if_expression(M_Ast *ast) {
    M_Token *first_token = token(ast);

    next_token(ast); // jump keyword

    if (token(ast) == NULL) {
        ast_error(ast, first_token, "unterminated if expression. expected an expression or a '{' but got EOF");
        synchronize(ast);
        return NULL;
    }

    M_Expression *condition = parse_expression_impl(ast);

    if (condition == NULL) {
        ast_error(ast, first_token, "unterminated if expression. expected an expression 'if ... {");
        synchronize(ast);
        return NULL;
    }

    if (token(ast) == NULL) {
        ast_error(ast, first_token, "unterminated if expression. expected '{' but got EOF");
        synchronize(ast);
        return NULL;
    }

    if (token(ast)->kind != M_LCURLY) {
        ast_error(ast, first_token, "unterminated if expression. expected '{' but got '%.*s'", token(ast)->size, token(ast)->value);
        synchronize(ast);
        return NULL;
    }

    next_token(ast); // skip '{'

    // @Leak TODO: maybe, use an arena. Just fix this leak. This block struct is never gonna be 'freed
    M_Expression_Block *block_head = malloc(sizeof(M_Expression_Block));
    block_head->next = NULL;
    block_head->expr = NULL;
    M_Expression_Block *block_tail = NULL;

    // Parse block
    while (token(ast) != NULL && token(ast)->kind != M_RCURLY) {
        if (token(ast) != NULL && token(ast)->kind == M_SEMI) {
            next_token(ast);
            continue;
        }

        M_Expression *expr = parse_expression_impl(ast);

        // just propagating upper errors
        if (expr == NULL) return NULL;

        if (block_tail == NULL) {
            block_head->expr = expr;
            block_head->next = NULL;
            block_tail = block_head;
        } else {
            M_Expression_Block *inner_block = malloc(sizeof(M_Expression_Block));

            inner_block->expr = expr;
            inner_block->next = NULL;

            block_tail->next = inner_block;
            block_tail = block_tail->next;
        }
    }

    if (token(ast) == NULL) {
        ast_error(ast, first_token, "unterminated if expression. expected '}' but got EOF");
        synchronize(ast);
        return NULL;
    }

    if (token(ast)->kind != M_RCURLY) {
        ast_error(ast, first_token, "unterminated if expression. expected '}' but got '%.*s", token(ast)->size, token(ast)->value);
        synchronize(ast);
        return NULL;
    }

    next_token(ast); // skip '}'

    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->kind = M_EK_IF;
    expr->if_expr.condition = condition;
    expr->if_expr.block = block_head;

    return expr;
}

static M_Expression *parse_primary_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    if (token(ast)->kind == M_NUMBER) {
        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
        expr->kind = M_EK_NUMBER;
        expr->number = convert_to_double(token(ast));

        next_token(ast);

        return expr;
    } else if (token(ast)->kind == M_ID) {
        // TODO: extract this to a list of keyword bindings where a name binds to a pointer
        // and this pointer is a function that parses this keyword semantics

        // WHILE
        if (token(ast)->size == 5 && strncmp(token(ast)->value, "while", 5) == 0) {
            return parse_loop_expression(ast);
        } else if (token(ast)->size == 5 && strncmp(token(ast)->value, "break", 5) == 0) {
            return parse_break_expression(ast);
        } else if (token(ast)->size == 2 && strncmp(token(ast)->value, "if", 2) == 0) {
            return parse_if_expression(ast);
        }

        if (checkahead(ast, M_LPAREN)) return parse_function_call_expression(ast);

        return parse_variable_expression(ast);
    } else if (token(ast)->kind == M_LPAREN) {
        M_Token *first_token = token(ast);

        next_token(ast);

        M_Expression *expr = parse_expression_impl(ast);

        if (token(ast) == NULL) {
            ast_error(ast, first_token, "unterminated parenthesis expression. expected ')' but got EOF");
            synchronize(ast);
            return NULL;
        }

        if (token(ast)->kind != M_RPAREN) {
            ast_error(ast, first_token, "unterminated parenthesis expression. expected ')' but got '%.*s'", token(ast)->size, token(ast)->value);
            synchronize(ast);
            return NULL;
        }

        next_token(ast);

        return expr;
    } else {
        ast_error(ast, token(ast), "expected number literal, function call or parenthesis expression but got '%.*s'", token(ast)->size, token(ast)->value);
        synchronize(ast);
        return NULL;
    }

    return NULL;
}

static M_Expression *parse_factorial_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_primary_expression(ast);

    if (left == NULL) return NULL;

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

    M_Token *first_token = token(ast);

    if (token(ast)->kind == M_MINUS) {
        M_Unary_Expression_Operator op = token_kind_as_unary_expression_operator(token(ast)->kind);

        next_token(ast);

        M_Expression *operand = parse_factorial_expression(ast);

        if (operand == NULL) {
            ast_error(ast, first_token, "missing operand for unary '%s'", unary_expression_operator_name(op));
            synchronize(ast);
            return NULL;
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

    if (left == NULL) return NULL;

    while (token(ast) != NULL && (token(ast)->kind == M_POW)) {
        M_Token *first_token = token(ast);

        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator(token(ast)->kind);

        next_token(ast);

        M_Expression *right = parse_power_expression(ast);

        if (right == NULL) {
            ast_error(ast, first_token, "missing right operand for '%s'", binary_expression_operator_name(op));
            synchronize(ast);
            return NULL;
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

    if (left == NULL) return NULL;

    while (token(ast) != NULL && (token(ast)->kind == M_TIMES || token(ast)->kind == M_DIVIDE || token(ast)->kind == M_MOD)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator(token(ast)->kind);

        M_Token *first_token = token(ast);

        next_token(ast);

        M_Expression *right = parse_power_expression(ast);

        if (right == NULL) {
            ast_error(ast, first_token, "missing right operand for '%s'", binary_expression_operator_name(op));
            synchronize(ast);
            return NULL;
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

static M_Expression *parse_additive_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_term_expression(ast);

    if (left == NULL) return NULL;

    while (token(ast) != NULL && (token(ast)->kind == M_PLUS || token(ast)->kind == M_MINUS)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator(token(ast)->kind);

        M_Token *first_token = token(ast);

        next_token(ast);

        M_Expression *right = parse_term_expression(ast);

        if (right == NULL) {
            ast_error(ast, first_token, "missing right operand for '%s'", binary_expression_operator_name(op));
            synchronize(ast);
            return NULL;
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

static M_Expression *parse_relational_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_additive_expression(ast);

    if (left == NULL) return NULL;

    while (token(ast) != NULL && (
               token(ast)->kind == M_LT || token(ast)->kind == M_LTE ||
               token(ast)->kind == M_GT || token(ast)->kind == M_GTE)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator(token(ast)->kind);

        M_Token *first_token = token(ast);
        next_token(ast);

        M_Expression *right = parse_additive_expression(ast);

        if (right == NULL) {
            ast_error(ast, first_token, "missing right operand for '%s'", binary_expression_operator_name(op));
            synchronize(ast);
            return NULL;
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

static M_Expression *parse_equality_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_relational_expression(ast);

    if (left == NULL) return NULL;

    while (token(ast) != NULL && (token(ast)->kind == M_EQUAL || token(ast)->kind == M_NOT_EQUAL)) {
        M_Binary_Expression_Operator op = token_kind_as_binary_expression_operator(token(ast)->kind);
        M_Token *first_token = token(ast);
        next_token(ast);

        M_Expression *right = parse_relational_expression(ast);

        if (right == NULL) {
            ast_error(ast, first_token, "missing right operand for '%s'", binary_expression_operator_name(op));
            synchronize(ast);
            return NULL;
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

static M_Expression *parse_assignment_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    if (token(ast)->kind == M_ID && checkahead(ast, M_ASSIGN)) {
        M_Token *first_token = token(ast);

        const char *name = token(ast)->value;
        int name_length = token(ast)->size;

        next_token(ast); // '='
        next_token(ast); // jumped '='

        M_Expression *right = parse_expression_impl(ast);

        if (right == NULL) {
            ast_error(ast, first_token, "missing right operand for assignment '%.*s = ...'", first_token->size, first_token->value);
            synchronize(ast);
            return NULL;
        }

        M_Expression *assignment_expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

        assignment_expr->kind = M_EK_ASSIGN;
        assignment_expr->assign.name.value = name;
        assignment_expr->assign.name.length = name_length;
        assignment_expr->assign.right = right;

        return assignment_expr;
    }

    return parse_equality_expression(ast);
}

static inline M_Expression *parse_expression_impl(M_Ast *ast) {
    return parse_assignment_expression(ast);
}

#define M_AST_MAX_EXPRESSION_ARRAY_SIZE 256

M_Ast *parse_expression(const char *filename, M_Token *head) {
    M_Ast *ast = malloc(sizeof(M_Ast));

    ast->errors = 0;
    ast->filename = filename;
    ast->current_token = head;
    ast->expressions_array_length = 0;
    ast->single_expression_arena = clibs_arena_create(sizeof(M_Expression) * 256, sizeof(M_Expression));;
    ast->expressions_array_arena = clibs_arena_create(sizeof(M_Expression*) * M_AST_MAX_EXPRESSION_ARRAY_SIZE, sizeof(M_Expression*));
    ast->expressions_array = (M_Expression**)ast->expressions_array_arena->buffer;

    ast->expressions_array[ast->expressions_array_length++] = parse_expression_impl(ast);

parse_expression_loop:
    while (token(ast) != NULL && token(ast)->kind == M_SEMI) {
        next_token(ast);

        if (ast->expressions_array_length >= M_AST_MAX_EXPRESSION_ARRAY_SIZE) {
            fprintf(stderr, "panic: you exceeded the maximum expressions list size of %d\n", M_AST_MAX_EXPRESSION_ARRAY_SIZE);
            ast_free(ast);
            exit(1);
        }

        M_Expression *expr = parse_expression_impl(ast);

        ast->expressions_array[ast->expressions_array_length++] = expr;
    }

    if (token(ast) != NULL) {
        ast_error(ast, token(ast), "expected EOF but got '%.*s'", token(ast)->size, token(ast)->value);
        synchronize(ast);

        goto parse_expression_loop;
    }

    if (ast->errors > 0) {
        fprintf(stderr, "compilation failed with \033[1;31m%ld\033[0m errors\n", ast->errors);

        ast_free(ast);

        return NULL;
    }

    return ast;
}

void ast_free(M_Ast *ast) {
    clibs_arena_destroy(ast->single_expression_arena);
    clibs_arena_destroy(ast->expressions_array_arena);

    free(ast);
}
