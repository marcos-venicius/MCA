#include "./lexer.h"

M_Lexer m_lexer_create(const char *content, const size_t content_size) {
    return (M_Lexer){
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

static inline void update_bot(M_Lexer *lexer) {
    lexer->bot = lexer->cursor;
}

static void advence_cursor(M_Lexer *lexer) {
    if (lexer->cursor < lexer->content_size) {
        lexer->cursor++;

        if (chr(lexer) == '\n') {
            lexer->col = 1;
            lexer->line++;
        } else {
            lexer->col++;
        }
    }
}

static void trim_whitespaces_and_line_breaks(M_Lexer *lexer) {
    while (chr(lexer) == ' ' || chr(lexer) == '\t' || chr(lexer) == '\r' || chr(lexer) == '\n') advence_cursor(lexer);
}

M_Token *m_lexer_tokenize(M_Lexer *lexer) {
    trim_whitespaces_and_line_breaks(lexer);
    update_bot(lexer);

    return lexer->head;
}
