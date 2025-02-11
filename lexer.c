#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include "./lexer.h"
#include "./log.h"

static inline char chr(M_Lexer *lexer);
static inline void advance_cursor(M_Lexer *lexer);

static size_t errors = 0;

static void unrecognized_symbol_error(M_Lexer *lexer) {
    errors++;

    if (lexer->filename) {
        fprintf(stderr, "%s:%ld:%ld: \033[1;31merror\033[0m unrecognized symbol \033[1;35m%c\033[0m\n",
                lexer->filename, lexer->line, lexer->col, chr(lexer));
    } else {
        fprintf(stderr, "%ld:%ld: \033[1;31merror\033[0m unrecognized symbol \033[1;35m%c\033[0m\n",
                lexer->line, lexer->col, chr(lexer));
    }

    advance_cursor(lexer);
}

static void invalid_floating_number_error(M_Lexer *lexer) {
    errors++;

    if (lexer->filename) {
        fprintf(stderr, "%s:%ld:%ld: \033[1;31merror\033[0m invalid floating number \033[1;35m%.*s\033[0m\n",
                lexer->filename, lexer->line, lexer->col, (int)(lexer->cursor - lexer->bot + 1), lexer->content + lexer->bot);
    } else {
        fprintf(stderr, "%ld:%ld: \033[1;31merror\033[0m invalid floating number \033[1;35m%.*s\033[0m\n",
                lexer->line, lexer->col, (int)(lexer->cursor - lexer->bot + 1), lexer->content + lexer->bot);
    }

    advance_cursor(lexer);
}

bool m_lexer_finished_with_errors() {
    return errors > 0;
}

const char *m_lexer_token_kind_display_name(M_Token_Kind kind) {
    switch (kind) {
        case M_NUMBER: return "NUMBER";
        case M_PLUS: return "PLUS";
        case M_DIVIDE: return "DIVIDE";
        case M_TIMES: return "TIMES";
        case M_MOD: return "MOD";
        case M_POW: return "POW";
        case M_MINUS: return "MINUS";
        case M_LPAREN: return "LPAREN";
        case M_RPAREN: return "RPAREN";
        default: return "<UNKOWN>";
    }
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

    if (lexer->head == NULL) {
        lexer->head = token;
        lexer->tail = token;
    } else {
        lexer->tail->next = token;
        lexer->tail = lexer->tail->next;
    }
}

static void tokenize_number(M_Lexer *lexer) {
    size_t digits = 0;

    while (is_digit(chr(lexer))) {
        advance_cursor(lexer);
        digits++;
    }

    if (chr(lexer) == '.') {
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

    save_token(lexer, M_NUMBER);
}

static void tokenize_single(M_Lexer *lexer) {
    switch (chr(lexer)) {
        case '*': { advance_cursor(lexer); save_token(lexer, M_TIMES); } break;
        case '/': { advance_cursor(lexer); save_token(lexer, M_DIVIDE); } break;
        case '(': { advance_cursor(lexer); save_token(lexer, M_LPAREN); } break;
        case ')': { advance_cursor(lexer); save_token(lexer, M_RPAREN); } break;
        case '+': { advance_cursor(lexer); save_token(lexer, M_PLUS); } break;
        case '-': { advance_cursor(lexer); save_token(lexer, M_MINUS); } break;
        case '%': { advance_cursor(lexer); save_token(lexer, M_MOD); } break;
        case '^': { advance_cursor(lexer); save_token(lexer, M_POW); } break;
        default: LOG("[!] unrecognized single token [%c]\n", chr(lexer)); return;
    }
}

M_Token *m_lexer_tokenize(M_Lexer *lexer) {
    while (chr(lexer) != '\0') {
        trim_whitespaces_and_line_breaks(lexer);
        update_bot(lexer);

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
            case '9': tokenize_number(lexer); break;
            case '*': tokenize_single(lexer); break;
            case '/': tokenize_single(lexer); break;
            case '(': tokenize_single(lexer); break;
            case ')': tokenize_single(lexer); break;
            case '+': tokenize_single(lexer); break;
            case '%': tokenize_single(lexer); break;
            case '^': tokenize_single(lexer); break;
            case '-': {
                if (is_digit(nchr(lexer))) {
                    advance_cursor(lexer);
                    tokenize_number(lexer);
                } else {
                    tokenize_single(lexer);
                }
            } break;
            case '\0': break;
            default: unrecognized_symbol_error(lexer); break;
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
