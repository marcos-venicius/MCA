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
