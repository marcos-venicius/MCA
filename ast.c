#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <assert.h>
#include <string.h>

#include "./ast.h"
#include "./arena.h"
#include "./lexer.h"
#include "./colors.h"
#include "constraints.h"

static M_Expression *parse_expression_impl(M_Ast *ast);
static M_Expression *parse_array_literal_expression(M_Ast *ast);
static M_Expression *parse_primary_expression(M_Ast *ast);
static M_Expression *parse_unary_expression(M_Ast *ast);

static inline M_Token *token(M_Ast *ast) {
    return ast->current_token;
}
static inline M_Token *ntoken(M_Ast *ast) {
    return ast->current_token->next;
}

static inline bool check(M_Ast *ast, M_Token_Kind kind) {
    return ast->current_token != NULL && ast->current_token->kind == kind;
}

static inline bool checkahead(M_Ast *ast, M_Token_Kind kind) {
    return ast->current_token != NULL && ast->current_token->next != NULL && ast->current_token->next->kind == kind;
}

static inline bool is_logical_and(M_Token *token) {
    return token != NULL && token->kind == M_ID && token->size == 3 && strncmp(token->value, "and", 3) == 0;
}

static inline bool is_logical_or(M_Token *token) {
    return token != NULL && token->kind == M_ID && token->size == 2 && strncmp(token->value, "or", 2) == 0;
}

static inline M_Token *next_token(M_Ast *ast) {
    if (ast->current_token != NULL) {
        ast->current_token = ast->current_token->next;

        if (ast->current_token) ast->last_consumed_token = ast->current_token;

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

static int64_t convert_to_integer(M_Token *token) {
    char buffer[token->size + 1];

    memcpy(buffer, token->value, token->size);
    buffer[token->size] = '\0';

    return strtoll(buffer, NULL, 10);
}

static M_Binary_Expression_Operator token_kind_as_binary_expression_operator(M_Token_Kind kind) {
    switch (kind) {
        case M_PLUS:        return M_BINARY_PLUS_OP;
        case M_TIMES:       return M_BINARY_TIMES_OP;
        case M_MINUS:       return M_BINARY_SUBTRACT_OP;
        case M_DIVIDE:      return M_BINARY_DIVIDE_OP;
        case M_MOD:         return M_BINARY_MOD_OP;
        case M_POW:         return M_BINARY_POW_OP;

        case M_EQUAL:       return M_BINARY_EQUAL_OP;
        case M_NOT_EQUAL:   return M_BINARY_NOT_EQUAL_OP;
        case M_GT:          return M_BINARY_GT_OP;
        case M_LT:          return M_BINARY_LT_OP;
        case M_GTE:         return M_BINARY_GTE_OP;
        case M_LTE:         return M_BINARY_LTE_OP;

        case M_LBRACKET:
        case M_RBRACKET:
        case M_PLUS_EQUAL:
        case M_MINUS_EQUAL:
        case M_QUESTION_MARK:
        case M_ASSIGN:
        case M_ID:
        case M_INT:
        case M_FLOAT:
        case M_STRING:
        case M_EXCLAMATION:
        case M_COLON:
        case M_BACKSLASH:
        case M_ARROW:
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

const char *binary_expression_operator_name(M_Binary_Expression_Operator op) {
    switch (op) {
        case M_BINARY_PLUS_OP:           return "+";
        case M_BINARY_TIMES_OP:          return "*";
        case M_BINARY_SUBTRACT_OP:       return "-";
        case M_BINARY_DIVIDE_OP:         return "/";
        case M_BINARY_MOD_OP:            return "%";
        case M_BINARY_POW_OP:            return "^";

        case M_BINARY_AND_OP:            return "and";
        case M_BINARY_OR_OP:             return "or";

        case M_BINARY_EQUAL_OP:          return "==";
        case M_BINARY_NOT_EQUAL_OP:      return "!=";
        case M_BINARY_GT_OP:             return ">";
        case M_BINARY_LT_OP:             return "<";
        case M_BINARY_GTE_OP:            return ">=";
        case M_BINARY_LTE_OP:            return "<=";

        case M_BINARY_OP_COUNT: break;
    }

    static_assert(M_BINARY_OP_COUNT == 14, "binary_expression_operator_name: unhandled binary operator");
    return NULL; // unreacheable
}

static const char *unary_expression_operator_name(M_Unary_Expression_Operator op) {
    switch (op) {
        case M_UNARY_MINUS_OP:      return "-";
        case M_UNARY_NOT_OP:
        case M_UNARY_FACTORIAL_OP:  return "!";
    }

    assert(0 && "unary_expression_operator_name: unreacheable");
}

static M_Unary_Expression_Operator token_kind_as_unary_expression_operator(M_Token_Kind kind) {
    switch (kind) {
        case M_MINUS:       return M_UNARY_MINUS_OP;
        case M_EXCLAMATION: return M_UNARY_FACTORIAL_OP;

        case M_STRING:
        case M_QUESTION_MARK:
        case M_ASSIGN:
        case M_PLUS:
        case M_PLUS_EQUAL:
        case M_MINUS_EQUAL:
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
        case M_INT:
        case M_FLOAT:
        case M_COLON:
        case M_BACKSLASH:
        case M_ARROW:
        case M_LPAREN:
        case M_RPAREN:
        case M_LCURLY:
        case M_RCURLY:
        case M_SEMI:
        case M_COMMA:
        case M_LBRACKET:
        case M_RBRACKET:
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

static void ast_info(M_Ast *ast, M_Token *token, const char *message, ...) {
    va_list args;
    va_start(args, message);

    if (ast->filename != NULL) {
        fprintf(stderr, "%s:%d:%d \033[0;36minfo\033[0m: ", ast->filename, token->loc.line, token->loc.col);
    } else {
        fprintf(stderr, "%d:%d \033[0;36minfo\033[0m: ", token->loc.line, token->loc.col);
    }

    vfprintf(stderr, message, args);
    fprintf(stderr, "\n");

    va_end(args);
}

static bool expect(M_Ast *ast, M_Token_Kind kind) {
    if (token(ast) == NULL) {
        ast_error(ast, ast->last_consumed_token, "expected '%s' but go EOF", m_lexer_token_kind_display_name(kind));
        synchronize(ast);
        return false;
    }

    if (token(ast)->kind != kind) {
        ast_error(ast, ast->last_consumed_token, "expected '%s' but go '%s'", m_lexer_token_kind_display_name(kind), m_lexer_token_kind_display_name(token(ast)->kind));
        synchronize(ast);
        return false;
    }

    return true;
}

static M_Expression *parse_identifier_expression(M_Ast *ast) {
    if (!expect(ast, M_ID)) return NULL;

    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->kind = M_EK_ID;
    expr->location = (M_Location){
        .line = token(ast)->loc.line,
        .col = token(ast)->loc.col,
        .filename = ast->filename
    };
    expr->Id.value = strndup(token(ast)->value, token(ast)->size);
    expr->Id.value_length = token(ast)->size;

    next_token(ast);

    return expr;
}

static M_Expression_Block *parse_block_expression(M_Ast *ast) {
    if (!token(ast)) {
        ast_error(ast, ast->last_consumed_token, "invalid block expression. expected '{' but got EOF");
        synchronize(ast);
        return NULL;
    }

    M_Expression_Block *block_head = NULL;
    M_Expression_Block *block_tail = NULL;

    if (token(ast)->kind == M_SEMI) {
        next_token(ast); // basically, an empty block 'while true;' '\(a, b) ->;', etc
    } else if (token(ast)->kind == M_LCURLY) {
        next_token(ast); // skip '{'

        // parsing the block
        while (token(ast) != NULL && token(ast)->kind != M_RCURLY) {
            // skip ';', empty expression (doesn't result on any value)
            if (token(ast)->kind == M_SEMI) {
                next_token(ast);
                continue;
            }

            M_Expression *expr = parse_expression_impl(ast);

            // @Urgent TODO: find a better way to propagate erros
            //               returning null may mean both propagating errors and an empty block;
            // just propagating upper errors
            if (expr == NULL) return NULL;

            if (block_tail == NULL) {
                block_head = clibs_arena_alloc(ast->block_expression_arena, sizeof(M_Expression_Block));
                block_head->expr = expr;
                block_head->next = NULL;
                block_tail = block_head;
            } else {
                M_Expression_Block *inner_block = clibs_arena_alloc(ast->block_expression_arena, sizeof(M_Expression_Block));

                inner_block->expr = expr;
                inner_block->next = NULL;

                block_tail->next = inner_block;
                block_tail = block_tail->next;
            }
        }

        if (token(ast) == NULL) {
            ast_error(ast, ast->last_consumed_token, "unterminated block expression. expected '{' but got EOF");
            synchronize(ast);
            return NULL;
        }

        if (token(ast)->kind != M_RCURLY) {
            ast_error(
                ast,
                ast->last_consumed_token,
                "unterminated block expression. expected '{' but got '%s'",
                m_lexer_token_kind_display_name(token(ast)->kind)
            );
            synchronize(ast);
            return NULL;
        }

        next_token(ast); // skip '}'
    } else {
        M_Expression *expr = parse_expression_impl(ast); // try to parse a single (inline) expression

        // propagating upper errors
        if (expr == NULL) return NULL;

        block_head = clibs_arena_alloc(ast->block_expression_arena, sizeof(M_Expression_Block));
        block_head->expr = expr;
        block_head->next = NULL;
    }

    return block_head;
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
    fn->location = (M_Location){
        .line = fn_name->loc.line,
        .col = fn_name->loc.col,
        .filename = ast->filename
    };
    fn->Call.fn_name.value = strndup(fn_name->value, fn_name->size);
    fn->Call.fn_name.value_length = fn_name->size;
    fn->Call.arguments_length = 0;

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

        fn->Call.arguments[fn->Call.arguments_length++] = expr;

        if (token(ast) == NULL) {
            ast_error(ast, lparen, "expected ',' or ')' but got EOF");
            synchronize(ast);
            return NULL;
        }

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

static M_Expression *parse_function_declaration_expression(M_Ast *ast) {
    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->kind = M_EK_FN;
    expr->location = (M_Location){
        .line = token(ast)->loc.line,
        .col = token(ast)->loc.col,
        .filename = ast->filename
    };
    expr->Fn.arguments_length = 0;

    next_token(ast); // skip '\'

    if (!expect(ast, M_LPAREN)) return NULL;
    next_token(ast); // skip '('

    // parse arguments
    while (token(ast) != NULL && token(ast)->kind != M_RPAREN) {
        if (!expect(ast, M_ID)) return NULL;

        M_Expression *arg_expr = parse_identifier_expression(ast);

        expr->Fn.arguments[expr->Fn.arguments_length++] = arg_expr;

        if (token(ast) == NULL) {
            ast_error(ast, ast->last_consumed_token, "expected ',' or ')' but got EOF");
            synchronize(ast);
            return NULL;
        }

        if (token(ast)->kind == M_RPAREN) break;

        if (token(ast)->kind != M_COMMA) {
            ast_error(ast, ast->last_consumed_token, "expected ',' but got '%s'", m_lexer_token_kind_display_name(token(ast)->kind));
            synchronize(ast);
            return NULL;
        }

        next_token(ast); // skip ','
    }

    if (!expect(ast, M_RPAREN)) return NULL;
    next_token(ast); // skip ')'

    if (!expect(ast, M_ARROW)) return NULL;
    next_token(ast); // skip '->'

    // TODO: handle error propagation
    expr->Fn.block = parse_block_expression(ast);

    return expr;
}

static M_Expression *parse_break_expression(M_Ast *ast) {
    M_Token *first_token = token(ast);

    next_token(ast); // jump 'break'

    M_Expression *break_value = NULL;

    if (token(ast) != NULL && token(ast)->kind != M_SEMI) {
        break_value = parse_expression_impl(ast);

        if (break_value == NULL) {
            ast_error(ast, first_token, "invalid break expression. Isn't it missing a ';'? 'break\033[1;35m;\033[0m'");
            ast_info(ast, first_token, "all break expressions that doesn't have a value should be terminated with a ';'");
            synchronize(ast);
            return NULL;
        }
    }

    M_Expression *loop_break = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

    loop_break->location = (M_Location){
        .line = first_token->loc.line,
        .col = first_token->loc.col,
        .filename = ast->filename
    };
    loop_break->kind = M_EK_BREAK;
    loop_break->Break = break_value;

    return loop_break;
}

static M_Expression *parse_return_expression(M_Ast *ast) {
    M_Token *first_token = token(ast);

    next_token(ast); // jump 'return'

    M_Expression *return_value = NULL;

    if (token(ast) != NULL && token(ast)->kind != M_SEMI && token(ast)->kind != M_RCURLY) {
        return_value = parse_expression_impl(ast);

        if (return_value == NULL) {
            ast_error(ast, first_token, "invalid return expression. Isn't it missing a ';'? 'return"C_MAGENTA";"C_RESET"'");
            ast_info(ast, first_token, "all return expressions that doesn't have a value should be terminated with a ';'");
            synchronize(ast);
            return NULL;
        }
    }

    M_Expression *fn_return = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

    fn_return->location = (M_Location){
        .line = first_token->loc.line,
        .col = first_token->loc.col,
        .filename = ast->filename
    };
    fn_return->kind = M_EK_RETURN;
    fn_return->Return = return_value;

    return fn_return;
}

static M_Expression *parse_while_expression(M_Ast *ast) {
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
    
    // TODO: check for error propagation
    M_Expression_Block *block = parse_block_expression(ast);


    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->kind = M_EK_WHILE;
    expr->location = (M_Location){
        .line = first_token->loc.line,
        .col = first_token->loc.col,
        .filename = ast->filename
    };
    expr->While.condition = condition;
    expr->While.block = block;

    return expr;
}

static M_Expression *parse_if_expression(M_Ast *ast) {
    M_Token *first_token = token(ast);

    next_token(ast); // jump 'if' keyword

    if (token(ast) == NULL) {
        ast_error(ast, first_token, "unterminated if expression. expected an expression but got EOF");
        synchronize(ast);
        return NULL;
    }

    M_Expression *if_condition = parse_expression_impl(ast);

    if (if_condition == NULL) {
        ast_error(ast, first_token, "unterminated if expression. expected an expression 'if ... {");
        synchronize(ast);
        return NULL;
    }

    if (token(ast) == NULL) {
        ast_error(ast, first_token, "unterminated if expression. expected '{' but got EOF");
        synchronize(ast);
        return NULL;
    }

    // TODO: check for error propagation
    M_Expression_Block *then_block = parse_block_expression(ast);

    M_Expression_Elif_Block *elif_head = NULL;
    M_Expression_Elif_Block *elif_tail = NULL;

    while (token(ast) != NULL && token(ast)->kind == M_ID && token(ast)->size == 4 && strncmp(token(ast)->value, "elif", 4) == 0) {
        if (elif_tail == NULL) {
            elif_head = clibs_arena_alloc(ast->block_expression_arena, sizeof(M_Expression_Elif_Block));
            elif_head->next = NULL;
            elif_head->condition = NULL;
            elif_head->block = NULL;
            elif_tail = elif_head;
        } else {
            M_Expression_Elif_Block *next_elif = clibs_arena_alloc(ast->block_expression_arena, sizeof(M_Expression_Elif_Block));
            next_elif->block = NULL;
            next_elif->condition = NULL;
            next_elif->next = NULL;

            elif_tail->next = next_elif;
            elif_tail = elif_tail->next;
        }

        next_token(ast); // skip 'elif' keyword

        if (token(ast) == NULL) {
            ast_error(ast, first_token, "unterminated elif expression. expected a condition 'elif ... {' but got EOF");
            synchronize(ast);
            return NULL;
        }

        elif_tail->condition = parse_expression_impl(ast);

        // just propagating upper errors
        if (elif_tail->condition == NULL) return NULL;

        // TODO: check for error propagation
        elif_tail->block = parse_block_expression(ast);
    }

    M_Expression_Block *else_block = NULL;

    if (token(ast) != NULL && token(ast)->kind == M_ID && token(ast)->size == 4 && strncmp(token(ast)->value, "else", 4) == 0) {
        next_token(ast); // jump 'else' keyword

        if (token(ast) == NULL) {
            ast_error(ast, first_token, "unterminated if expression. expected 'else' block but got EOF");
            synchronize(ast);
            return NULL;
        }

        // TODO: check for error propagation
        else_block = parse_block_expression(ast);
    }

    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->kind = M_EK_IF;
    expr->location = (M_Location){
        .line = first_token->loc.line,
        .col = first_token->loc.col,
        .filename = ast->filename
    };
    expr->If.condition = if_condition;
    expr->If.then_block = then_block;
    expr->If.elif_blocks = elif_head;
    expr->If.else_block = else_block;

    return expr;
}

static M_Expression *parse_boolean_expression(M_Ast *ast, bool value) {
    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->location = (M_Location){
        .line = token(ast)->loc.line,
        .col = token(ast)->loc.col,
        .filename = ast->filename
    };
    expr->kind = M_EK_BOOL;
    expr->Bool = value;

    next_token(ast);

    return expr;
}

static M_Expression *parse_string_literal_expression(M_Ast *ast) {
    M_Token *tk = token(ast);

    char *string_literal = malloc(tk->size + 1);

    int tk_str_index = 0;
    int str_literal_index = 0;

    while (tk_str_index < tk->size) {
        if (tk->value[tk_str_index] == '\\') {
            if (tk_str_index + 1 >= tk->size) break;

            switch (tk->value[tk_str_index + 1]) {
                case '\'':
                    string_literal[str_literal_index] = '\'';
                    break;
                case 'n':
                    string_literal[str_literal_index] = '\n';
                    break;
                case '\\':
                    string_literal[str_literal_index] = '\\';
                    break;
                default:
                    // If it's an unrecognized escape, just copy the character
                    string_literal[str_literal_index] = tk->value[tk_str_index + 1];
                    break;
            }

            tk_str_index++; // Skip the backslash
        } else {
            string_literal[str_literal_index] = tk->value[tk_str_index];
        }

        str_literal_index++;
        tk_str_index++;
    }

    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->location = (M_Location){
        .line = tk->loc.line,
        .col = tk->loc.col,
        .filename = ast->filename
    };
    
    expr->kind = M_EK_STRING;
    expr->String.value = string_literal;
    expr->String.value_length = str_literal_index;
    expr->String.value[str_literal_index] = '\0';

    next_token(ast); // Jump over the string token

    return expr;
}

static M_Expression *parse_map_expression(M_Ast *ast) {
    next_token(ast); // skip '}'

    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->kind = M_EK_MAP;
    expr->location = (M_Location){
        .line = token(ast)->loc.line,
        .col = token(ast)->loc.col,
        .filename = ast->filename
    };
    expr->Map.items_length = 0;
    expr->Map.items = NULL;

    int capacity = 8;
    M_Expression **temp_items = malloc(sizeof(M_Expression *) * capacity);
    int length = 0;

    while (!check(ast, M_RCURLY)) {
        M_Token *key_start_token = token(ast);
        M_Expression *key = parse_unary_expression(ast);

        if (!check(ast, M_COLON)) {
            free(temp_items);
            ast_error(ast, key_start_token, "missing value for key inside the map");
            synchronize(ast);
            return NULL;
        }

        next_token(ast); // skip ':'

        M_Expression *value = parse_unary_expression(ast);

        if (value == NULL) {
            free(temp_items);
            return NULL;
        }

        if (check(ast, M_COMMA)) {
            next_token(ast);
        } else if (!check(ast, M_RCURLY)) {
            free(temp_items);
            ast_error(ast, ast->last_consumed_token, "expected ',' but got unexpected '%s'", m_lexer_token_kind_display_name(ast->last_consumed_token->kind));
            synchronize(ast);
            return NULL;
        }

        if (length >= capacity) {
            capacity *= 2;
            temp_items = realloc(temp_items, sizeof(M_Expression *) * capacity);
        }

        temp_items[length++] = key;
        temp_items[length++] = value;
    }

    if (!check(ast, M_RCURLY)) {
        free(temp_items);
        ast_error(ast, ast->last_consumed_token, "unclosed curly expression '{...'");
        synchronize(ast);
        return NULL;
    }

    next_token(ast); // skip '}'

    if (length > 0) {
        expr->Map.items = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression *) * length);
        memcpy(expr->Map.items, temp_items, sizeof(M_Expression *) * length);
    }
    expr->Map.items_length = length;

    free(temp_items);

    return expr;
}

static M_Expression *parse_primary_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    if (token(ast)->kind == M_INT) {
        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
        expr->location = (M_Location){
            .line = token(ast)->loc.line,
            .col = token(ast)->loc.col,
            .filename = ast->filename
        };
        expr->kind = M_EK_INT;
        expr->Int = convert_to_integer(token(ast));

        next_token(ast);

        return expr;
    } else if (token(ast)->kind == M_QUESTION_MARK) {
        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
        expr->location = (M_Location){
            .line = token(ast)->loc.line,
            .col = token(ast)->loc.col,
            .filename = ast->filename
        };
        expr->kind = M_EK_UNIT;

        next_token(ast);

        return expr;
    } else if (token(ast)->kind == M_FLOAT) {
        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
        expr->location = (M_Location){
            .line = token(ast)->loc.line,
            .col = token(ast)->loc.col,
            .filename = ast->filename
        };
        expr->kind = M_EK_FLOAT;
        expr->Float = convert_to_double(token(ast));

        next_token(ast);

        return expr;
    } else if (token(ast)->kind == M_ID) {
        // TODO: extract this to a list of keyword bindings where a name binds to a pointer
        // and this pointer is a function that parses this keyword semantics

        if (token(ast)->size == 5 && strncmp(token(ast)->value, "while", 5) == 0) {
            return parse_while_expression(ast);
        } else if (token(ast)->size == 5 && strncmp(token(ast)->value, "break", 5) == 0) {
            return parse_break_expression(ast);
        } else if (token(ast)->size == 6 && strncmp(token(ast)->value, "return", 6) == 0) {
            return parse_return_expression(ast);
        } else if (token(ast)->size == 2 && strncmp(token(ast)->value, "if", 2) == 0) {
            return parse_if_expression(ast);
        } else if (token(ast)->size == 4 && strncmp(token(ast)->value, "true", 4) == 0) {
            return parse_boolean_expression(ast, true);
        } else if (token(ast)->size == 5 && strncmp(token(ast)->value, "false", 5) == 0) {
            return parse_boolean_expression(ast, false);
        }

        if (checkahead(ast, M_LPAREN)) return parse_function_call_expression(ast);

        return parse_identifier_expression(ast);
    } else if (token(ast)->kind == M_STRING) {
        return parse_string_literal_expression(ast);
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
    } else if (token(ast)->kind == M_BACKSLASH) {
        return parse_function_declaration_expression(ast);
    } else if (token(ast)->kind == M_LBRACKET) {
        return parse_array_literal_expression(ast);
    } else if (token(ast)->kind == M_LCURLY) {
        return parse_map_expression(ast);
    } else {
        ast_error(ast, token(ast), "expected number literal, function call or parenthesis expression but got '%.*s'", token(ast)->size, token(ast)->value);
        synchronize(ast);
        return NULL;
    }

    return NULL;
}

static M_Expression *parse_array_literal_expression(M_Ast *ast) {
    M_Token *start_token = token(ast);
    next_token(ast); // skip '['
    
    M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
    expr->kind = M_EK_ARRAY;
    expr->location = (M_Location){
        .line = start_token->loc.line,
        .col = start_token->loc.col,
        .filename = ast->filename
    };
    expr->Array.items_length = 0;
    expr->Array.items = NULL;
    
    int capacity = 8;
    M_Expression **temp_items = malloc(sizeof(M_Expression *) * capacity);
    int length = 0;

    if (token(ast) != NULL && token(ast)->kind != M_RBRACKET) {
        while (true) {
            M_Expression *item = parse_expression_impl(ast);
            if (!item) {
                free(temp_items);
                return NULL;
            }
            
            if (length >= capacity) {
                capacity *= 2;
                temp_items = realloc(temp_items, sizeof(M_Expression *) * capacity);
            }
            temp_items[length++] = item;
            
            if (token(ast) == NULL) {
                ast_error(ast, start_token, "expected ']' but got EOF");
                synchronize(ast);
                free(temp_items);
                return NULL;
            }
            if (token(ast)->kind == M_RBRACKET) break;
            if (token(ast)->kind != M_COMMA) {
                ast_error(ast, token(ast), "expected ',' or ']'");
                synchronize(ast);
                free(temp_items);
                return NULL;
            }
            next_token(ast); // skip ','
        }
    }
    
    if (length > 0) {
        expr->Array.items = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression *) * length);
        memcpy(expr->Array.items, temp_items, sizeof(M_Expression *) * length);
    }
    expr->Array.items_length = length;
    free(temp_items);
    
    if (!expect(ast, M_RBRACKET)) return NULL;
    next_token(ast); // skip ']'
    
    return expr;
}

static M_Expression *parse_postfix_expression(M_Ast *ast) {
    M_Expression *left = parse_primary_expression(ast);
    if (left == NULL) return NULL;

    while (token(ast) != NULL && token(ast)->kind == M_LBRACKET) {
        M_Token *bracket_token = token(ast);
        next_token(ast); // skip '['

        M_Expression *index = parse_expression_impl(ast);

        if (!expect(ast, M_RBRACKET)) return NULL;
        next_token(ast); // skip ']'

        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));
        expr->kind = M_EK_INDEX;
        expr->location = (M_Location){
            .line = bracket_token->loc.line,
            .col = bracket_token->loc.col,
            .filename = ast->filename
        };
        expr->Index.left = left;
        expr->Index.index = index;

        left = expr;
    }
    return left;
}

static M_Expression *parse_factorial_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_postfix_expression(ast);

    if (left == NULL) return NULL;

    while (token(ast) != NULL && token(ast)->kind == M_EXCLAMATION) {
        M_Token *op_token = token(ast);

        M_Unary_Expression_Operator op = token_kind_as_unary_expression_operator(token(ast)->kind);

        next_token(ast);

        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

        expr->kind = M_EK_UNARY;
        expr->location = (M_Location){
            .line = op_token->loc.line,
            .col = op_token->loc.col,
            .filename = ast->filename
        };
        expr->Unary.op = op;
        expr->Unary.operand = left;

        left = expr;
    }

    return left;
}

static M_Expression *parse_unary_expression(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Token *first_token = token(ast);

    if (token(ast)->kind == M_MINUS || token(ast)->kind == M_EXCLAMATION) {
        M_Token *op_token = token(ast);
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
        expr->location = (M_Location){
            .line = op_token->loc.line,
            .col = op_token->loc.col,
            .filename = ast->filename
        };
        expr->Unary.op = op == M_UNARY_FACTORIAL_OP ? M_UNARY_NOT_OP : op;
        expr->Unary.operand = operand;

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
        expr->location = (M_Location){
            .line = first_token->loc.line,
            .col = first_token->loc.col,
            .filename = ast->filename
        };
        expr->Binary.op = op;
        expr->Binary.left = left;
        expr->Binary.right = right;

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
        // TODO: should this be the location's operator or the first token of the expression?
        expr->location = (M_Location){
            .line = first_token->loc.line,
            .col = first_token->loc.col,
            .filename = ast->filename
        };
        expr->Binary.op = op;
        expr->Binary.left = left;
        expr->Binary.right = right;

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
        // TODO: should it be the location of the operator or the start of the expression?
        expr->location = (M_Location){
            .line = first_token->loc.line,
            .col = first_token->loc.col,
            .filename = ast->filename
        };
        expr->Binary.op = op;
        expr->Binary.left = left;
        expr->Binary.right = right;

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
        // TODO: should it be the location of the operator or the first token of expression
        expr->location = (M_Location){
            .line = first_token->loc.line,
            .col = first_token->loc.col,
            .filename = ast->filename
        };
        expr->Binary.op = op;
        expr->Binary.left = left;
        expr->Binary.right = right;

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
        // TODO: should this be the operator location or the start of the expression?
        expr->location = (M_Location){
            .line = first_token->loc.line,
            .col = first_token->loc.col,
            .filename = ast->filename
        };
        expr->Binary.op = op;
        expr->Binary.left = left;
        expr->Binary.right = right;

        left = expr;
    }

    return left;
}

static M_Expression *parse_logical_operators(M_Ast *ast) {
    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_equality_expression(ast);

    bool and = false;

    while (token(ast) != NULL && ((and = is_logical_and(token(ast))) || is_logical_or(token(ast)))) {
        M_Token *first_token = token(ast);
        next_token(ast);

        M_Expression *right = parse_equality_expression(ast);

        if (right == NULL) {
            ast_error(ast, first_token, "missing right operand for '%s' operator", and ? "and" : "or");
            synchronize(ast);
            return NULL;
        }

        M_Expression *expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

        expr->kind = M_EK_BINARY;
        // TODO: should be the location of the operator or the location of the start of the binary expression?
        expr->location = (M_Location){
            .line = first_token->loc.line,
            .col = first_token->loc.col,
            .filename = ast->filename
        };
        expr->Binary.op = and ? M_BINARY_AND_OP : M_BINARY_OR_OP;
        expr->Binary.left = left;
        expr->Binary.right = right;

        left = expr;
    }

    return left;
}

static M_Expression *parse_assignment_expression_impl(M_Ast *ast, M_Expression_Kind kind, M_Expression *left) {
    M_Token *first_token = token(ast);

    next_token(ast); // skip '=', '-=', '+='

    M_Expression *right = parse_expression_impl(ast);

    if (right == NULL) return NULL;

    M_Expression *assignment_expr = clibs_arena_alloc(ast->single_expression_arena, sizeof(M_Expression));

    assignment_expr->kind = kind;
    assignment_expr->location = (M_Location){
        .line = first_token->loc.line,
        .col = first_token->loc.col,
        .filename = ast->filename
    };
    assignment_expr->Assign.left  = left;
    assignment_expr->Assign.right = right;

    return assignment_expr;
}

static M_Expression *parse_assignment_expression(M_Ast *ast) {
    M_Token *first_token = token(ast);

    if (token(ast) == NULL) return NULL;

    M_Expression *left = parse_logical_operators(ast);

    if (left != NULL && (left->kind & (ACCEPTABLE_ASSIGNMENT_LEFT_SIDE_EXPRESSION_KINDS)) && (check(ast, M_ASSIGN) || check(ast, M_PLUS_EQUAL) || check(ast, M_MINUS_EQUAL))) {
        M_Token *op_token = token(ast);

        static M_Expression_Kind op[M_ASSIGN + M_MINUS_EQUAL + M_PLUS_EQUAL + 1] = {
            [M_ASSIGN]      = M_EK_ASSIGN,
            [M_PLUS_EQUAL]  = M_EK_ADD_ASSIGN,
            [M_MINUS_EQUAL] = M_EK_SUB_ASSIGN
        };

        static const char *messages[M_ASSIGN + M_MINUS_EQUAL + M_PLUS_EQUAL + 1] = {
            [M_ASSIGN]      = "missing right operand for assignment '%.*s = ...'",
            [M_PLUS_EQUAL]  = "missing right operand for addition assignment '%.*s += ...'",
            [M_MINUS_EQUAL] = "missing right operand for subtraction assignment '%.*s -= ...'"
        };

        M_Expression *expr = parse_assignment_expression_impl(ast, op[op_token->kind], left);

        if (expr == NULL) {
            ast_error(ast, first_token, messages[op_token->kind], first_token->size, first_token->value);
            synchronize(ast);
            return NULL;
        }

        return expr;
    }

    return left;
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
    ast->last_consumed_token = head;
    ast->expressions_array_length = 0;
    // TODO: I have these values fixed, maybe in the future we should make them grow (inside the arena library)
    ast->block_expression_arena = clibs_arena_create(sizeof(M_Expression_Block) * 256, sizeof(M_Expression_Block));
    ast->single_expression_arena = clibs_arena_create(sizeof(M_Expression) * 256, sizeof(M_Expression));
    ast->expressions_array_arena = clibs_arena_create(sizeof(M_Expression*) * M_AST_MAX_EXPRESSION_ARRAY_SIZE, sizeof(M_Expression*));
    ast->expressions_array = (M_Expression**)ast->expressions_array_arena->buffer;

parse_expression_loop:
    while (token(ast) != NULL) {
        if (token(ast)->kind == M_SEMI) {
            next_token(ast);
            continue;
        }

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

    if (ast->errors > 0)
        fprintf(stderr, "compilation failed with \033[1;31m%ld\033[0m errors\n", ast->errors);

    return ast;
}

void ast_free(M_Ast *ast) {
    clibs_arena_destroy(ast->single_expression_arena);
    clibs_arena_destroy(ast->expressions_array_arena);
    clibs_arena_destroy(ast->block_expression_arena);

    free(ast);
}
