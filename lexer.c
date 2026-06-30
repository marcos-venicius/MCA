#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <assert.h>
#include <stdarg.h>

#include "./lexer.h"

static inline char chr(M_Lexer *lexer);
static inline void advance_cursor(M_Lexer *lexer);

static size_t errors = 0;

static void m_lexer_error(M_Lexer *lexer, const char *fmt, ...) {
    errors++;

    va_list args;
    va_start(args, fmt);

    if (lexer->filename) {
        fprintf(stderr, "%s:%ld:%ld: \033[1;31merror\033[0m ", lexer->filename, lexer->line, lexer->col);
    } else {
        fprintf(stderr, "%ld:%ld: \033[1;31merror\033[0m ", lexer->line, lexer->col);
    }

    vfprintf(stderr, fmt, args);

    fprintf(stderr, "\n");

    advance_cursor(lexer);
    va_end(args);
}

static void unrecognized_symbol_error(M_Lexer *lexer) {
    m_lexer_error(lexer, "unrecognized symbol \033[1;35m%c\033[0m", lexer->col, chr(lexer));

    advance_cursor(lexer);
}

static void invalid_floating_number_error(M_Lexer *lexer) {
    m_lexer_error(lexer, "invalid floating number \033[1;35m%.*s\033[0m", (int)(lexer->cursor - lexer->bot + 1), lexer->content + lexer->bot);

    advance_cursor(lexer);
}

bool m_lexer_finished_with_errors() {
    return errors > 0;
}

const char *m_lexer_token_kind_display_name(M_Token_Kind kind) {
    switch (kind) {
        case M_INT: return "int";
        case M_FLOAT: return "float";
        case M_STRING: return "string";

        case M_ID: return "identifier";

        case M_PLUS: return "+";
        case M_PLUS_EQUAL: return "+=";
        case M_DIVIDE: return "/";
        case M_TIMES: return "*";
        case M_MOD: return "%";
        case M_POW: return "^";
        case M_MINUS: return "-";
        case M_MINUS_EQUAL: return "-=";

        case M_EXCLAMATION: return "!";

        case M_QUESTION_MARK: return "?";

        case M_ASSIGN: return "=";

        case M_EQUAL: return "==";
        case M_NOT_EQUAL: return "!=";
        case M_GT: return ">";
        case M_LT: return "<";
        case M_GTE: return ">=";
        case M_LTE: return "<=";

        case M_LPAREN: return "(";
        case M_RPAREN: return ")";
        case M_LCURLY: return "{";
        case M_RCURLY: return "}";
        case M_COMMA: return ",";
        case M_SEMI: return ";";
    }

    assert(0 && "m_lexer_token_kind_display_name: unhandled M_Token_Kind case");
}

M_Lexer m_lexer_create(const char *filename, const char *content, const size_t content_size) {
    return (M_Lexer){
        .filename = filename,
        .content = content,
        .content_size = content_size,
        .line = 1,
        .col = 1,
        .bot = 0,
        .tail = NULL,
        .head = NULL
    };
}

static inline bool is_identifier_start(char c) {
    return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_';
}

static inline bool keep_being_identifier(char c) {
    return is_identifier_start(c) || (c >= '0' && c <= '9');
}

static inline char chr(M_Lexer *lexer) {
    return lexer->cursor < lexer->content_size ? lexer->content[lexer->cursor] : '\0';
}

static inline char nchr(M_Lexer *lexer) {
    size_t offset = 1;

    return lexer->cursor < lexer->content_size + offset ? lexer->content[lexer->cursor + offset] : '\0';
}

static inline void update_bot(M_Lexer *lexer) {
    lexer->bot = lexer->cursor;
}

static inline void advance_cursor(M_Lexer *lexer) {
    if (lexer->cursor < lexer->content_size) {
        if (chr(lexer) == '\n') {
            lexer->col = 1;
            lexer->line++;
        } else {
            lexer->col++;
        }

        lexer->cursor++;
    }
}

static inline void trim_whitespaces_and_line_breaks(M_Lexer *lexer) {
    while (chr(lexer) == ' ' || chr(lexer) == '\t' || chr(lexer) == '\r' || chr(lexer) == '\n') advance_cursor(lexer);
}

static inline bool is_digit(char c) {
    return c >= '0' && c <= '9';
}

static void append_token(M_Lexer *lexer, M_Token *token) {
    if (lexer->head == NULL) {
        lexer->head = token;
        lexer->tail = token;
    } else {
        lexer->tail->next = token;
        lexer->tail = lexer->tail->next;
    }
}

static void save_token(M_Lexer *lexer, M_Token_Kind kind) {
    M_Token *token = malloc(sizeof(M_Token));

    if (token == NULL) {
        fprintf(stderr, "could not allocate memory for the token [%.*s]: %s\n",
                (int)(lexer->cursor - lexer->bot), lexer->content + lexer->bot, strerror(errno));
        exit(1);
    }

    token->kind = kind;
    token->value = lexer->content + lexer->bot;
    token->size = lexer->cursor - lexer->bot;
    token->next = NULL;
    token->loc.line = lexer->tok_line;
    token->loc.col = lexer->tok_col;

    append_token(lexer, token);
}

static void save_token_parametrized(M_Lexer *lexer, M_Token_Kind kind, int bot, int cursor) {
    M_Token *token = malloc(sizeof(M_Token));

    if (token == NULL) {
        fprintf(stderr, "could not allocate memory for the token [%.*s]: %s\n",
                (int)(lexer->cursor - lexer->bot), lexer->content + lexer->bot, strerror(errno));
        exit(1);
    }

    token->kind = kind;
    token->value = lexer->content + bot;
    token->size = cursor - bot;
    token->next = NULL;
    token->loc.line = lexer->tok_line;
    token->loc.col = lexer->tok_col;

    append_token(lexer, token);
}

static void tokenize_number(M_Lexer *lexer) {
    size_t digits = 0;

    while (is_digit(chr(lexer))) {
        advance_cursor(lexer);
        digits++;
    }

    M_Token_Kind kind = M_INT;

    if (chr(lexer) == '.') {
        kind = M_FLOAT;

        if (digits == 0) {
            invalid_floating_number_error(lexer);
            return;
        }

        if (!is_digit(nchr(lexer))) {
            invalid_floating_number_error(lexer);
            return;
        }

        advance_cursor(lexer);

        while (is_digit(chr(lexer))) advance_cursor(lexer);
    }

    save_token(lexer, kind);
}

static void tokenize_string(M_Lexer *lexer) {
    advance_cursor(lexer); // skip "'"

    while (chr(lexer) != '\0' && chr(lexer) != '\'') {
        if (chr(lexer) == '\n') {
            m_lexer_error(lexer, "you cannot have line breaks inside literal strings");
        }

        if (chr(lexer) == '\\') {
            switch (nchr(lexer)) {
                case '\\':
                    advance_cursor(lexer);
                    break;
                case '\'':
                    advance_cursor(lexer);
                    break;
                case 'n':
                    advance_cursor(lexer);
                    break;
                default:
                    m_lexer_error(lexer, "invalid scaping sequence '\\%c'", nchr(lexer));
                    break;
            }
        }

        advance_cursor(lexer);
    }

    if (chr(lexer) != '\'')
        m_lexer_error(lexer, "unterminated string literal"); // we have errors, just don't tokenize this
    else {
        advance_cursor(lexer); // skip "'"

        save_token_parametrized(lexer, M_STRING, lexer->bot + 1, lexer->cursor - 1);
    }
}

static void tokenize_single(M_Lexer *lexer) {
    switch (chr(lexer)) {
        case '*': { advance_cursor(lexer); save_token(lexer, M_TIMES); } break;
        case '/': { advance_cursor(lexer); save_token(lexer, M_DIVIDE); } break;
        case '(': { advance_cursor(lexer); save_token(lexer, M_LPAREN); } break;
        case ')': { advance_cursor(lexer); save_token(lexer, M_RPAREN); } break;
        case '{': { advance_cursor(lexer); save_token(lexer, M_LCURLY); } break;
        case '}': { advance_cursor(lexer); save_token(lexer, M_RCURLY); } break;
        case '+': { advance_cursor(lexer); save_token(lexer, M_PLUS); } break;
        case '-': { advance_cursor(lexer); save_token(lexer, M_MINUS); } break;
        case '%': { advance_cursor(lexer); save_token(lexer, M_MOD); } break;
        case '^': { advance_cursor(lexer); save_token(lexer, M_POW); } break;
        case '!': { advance_cursor(lexer); save_token(lexer, M_EXCLAMATION); } break;
        case ';': { advance_cursor(lexer); save_token(lexer, M_SEMI); } break;
        case ',': { advance_cursor(lexer); save_token(lexer, M_COMMA); } break;
        case '?': { advance_cursor(lexer); save_token(lexer, M_QUESTION_MARK); } break;
        default: m_lexer_error(lexer, "unrecognized single token %c\n", chr(lexer)); return;
    }
}

static void tokenize_n(M_Lexer *lexer, int n, M_Token_Kind kind) {
    for (int i = 0; i < n; i++) advance_cursor(lexer);

    save_token(lexer, kind);
}

static void tokenize_identifier(M_Lexer *lexer) {
    while (keep_being_identifier(chr(lexer))) advance_cursor(lexer);

    save_token(lexer, M_ID);
}

static void skip_comment(M_Lexer *lexer) {
    while (chr(lexer) != '\0' && chr(lexer) != '\n') advance_cursor(lexer);

    advance_cursor(lexer);
}

M_Token *m_lexer_tokenize(M_Lexer *lexer) {
    while (chr(lexer) != '\0') {
        trim_whitespaces_and_line_breaks(lexer);
        update_bot(lexer);

        lexer->tok_col = lexer->col;
        lexer->tok_line = lexer->line;

        switch (chr(lexer)) {
            case '0':
            case '1':
            case '2':
            case '3':
            case '4':
            case '5':
            case '6':
            case '7':
            case '8':
            case '9':
                tokenize_number(lexer);
                break;
            case '\'':
                tokenize_string(lexer);
                break;
            case '*':
            case '/':
            case '(':
            case ')':
            case '{':
            case '}':
            case '%':
            case '^':
            case ';':
            case ',':
            case '?':
                tokenize_single(lexer);
                break;
            case '-':
                if (nchr(lexer) == '=') {
                    tokenize_n(lexer, 2, M_MINUS_EQUAL);
                } else {
                    tokenize_single(lexer);
                }
                break;
            case '+':
                if (nchr(lexer) == '=') {
                    tokenize_n(lexer, 2, M_PLUS_EQUAL);
                } else {
                    tokenize_single(lexer);
                }
                break;
            case '!':
                if (nchr(lexer) == '=') {
                    tokenize_n(lexer, 2, M_NOT_EQUAL);
                } else {
                    tokenize_single(lexer);
                }
                break;
            case '=':
                if (nchr(lexer) == '=') {
                    tokenize_n(lexer, 2, M_EQUAL);
                } else {
                    tokenize_n(lexer, 1, M_ASSIGN);
                }
                break;
            case '<':
                if (nchr(lexer) == '=') {
                    tokenize_n(lexer, 2, M_LTE);
                } else {
                    tokenize_n(lexer, 1, M_LT);
                }
                break;
            case '>':
                if (nchr(lexer) == '=') {
                    tokenize_n(lexer, 2, M_GTE);
                } else {
                    tokenize_n(lexer, 1, M_GT);
                }
                break;
            case '#': skip_comment(lexer); break;
            case '\0': break;
            default: {
                if (is_identifier_start(chr(lexer))) {
                    tokenize_identifier(lexer);
                } else {
                    unrecognized_symbol_error(lexer);
                }
             } break;
        }
    }

    if (errors > 0) {
        m_lexer_free(lexer);

        return NULL;
    }

    return lexer->head;
}

void m_lexer_free(M_Lexer *lexer) {
    M_Token *current = lexer->head;

    while (current != NULL) {
        M_Token *next = current->next;

        free(current);

        current = next;
    }

    lexer->head = NULL;
    lexer->tail = NULL;
}
