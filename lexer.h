#ifndef LEXER_H_
#define LEXER_H_

#include <stddef.h>
#include <stdbool.h>

typedef struct M_Token M_Token;

typedef enum {
    // all numbers will be handled as C doubles
    M_NUMBER = 0,

    // signs
    M_PLUS,
    M_DIVIDE,
    M_TIMES,
    M_MINUS,
    M_MOD,
    M_POW,
    M_FACTORIAL,

    // symbols
    M_LPAREN,
    M_RPAREN,
    M_SEMI,
} M_Token_Kind;

struct M_Token {
    M_Token_Kind kind;
    const char *value;
    size_t size;

    M_Token *next;

    struct {
        int col, line;
    } loc;
};

typedef struct {
    const char *filename;
    const char *content;
    const size_t content_size;
    size_t cursor, bot, line, col;

    // those are the column and the line
    // at the start of the token
    size_t tok_col, tok_line;

    M_Token *head;
    M_Token *tail;
} M_Lexer;

bool m_lexer_finished_with_errors();
const char *m_lexer_token_kind_display_name(M_Token_Kind kind);
M_Lexer m_lexer_create(const char *filename, const char *content, const size_t content_size);
M_Token *m_lexer_tokenize(M_Lexer *lexer);
void m_lexer_free(M_Lexer *lexer);

#endif // LEXER_H_
