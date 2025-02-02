#ifndef LEXER_H_
#define LEXER_H_

#include <stddef.h>


typedef struct M_Token M_Token;

typedef enum {
    // all numbers will be handled as C doubles
    M_NUMBER = 0,

    // signs
    M_PLUS,
    M_DIVIDE,
    M_TIMES,
    M_MINUS,

    // symbols
    M_LPAREN,
    M_RPAREN,
} M_Token_Kind;

struct M_Token {
    M_Token_Kind kind;
    char *value;
    size_t size;

    M_Token *next;
};

typedef struct {
    const char *content;
    const size_t content_size;
    size_t cursor, bot, line, col;

    M_Token *head;
    M_Token *tail;
} M_Lexer;

M_Lexer m_lexer_create(const char *content, const size_t content_size);
M_Token *m_lexer_tokenize(M_Lexer *lexer);

#endif // LEXER_H_
