#include "./io.h"
#include "./interpreter.h"

#include <string.h>

typedef struct {
    const char *input_file_name;
    const char *argv[256];
    int argc;
} ProgramArguments;

void usage(FILE *stream, const char *program_name) {
    fprintf(stream, "USAGE: %s <file> [argv]\n\n", program_name);
    fprintf(stream, "    -h                  show this help\n");
    fprintf(stream, "\n");
}

bool cmp_arg(const char *a, const char *b) {
    size_t sa = strlen(a);
    size_t sb = strlen(b);

    if (sa != sb) return false;

    for (size_t i = 0; i < sa; ++i)
        if (a[i] != b[i]) return false;

    return true;
}

const char *shift(int *argc, char ***argv)
{
    if (*argc == 0) return NULL;

    const char *result = *argv[0];
    *argc -= 1;
    *argv += 1;
    return result;
}

int main(int argc, char **argv) {
    const char *program_name = shift(&argc, &argv);

    ProgramArguments p_arguments = {0};

    const char *arg = shift(&argc, &argv);

    while (arg != NULL) {
        if (cmp_arg(arg, "-h")) {
            usage(stdout, program_name);
            return 0;
        } else if (p_arguments.input_file_name == NULL) {
            p_arguments.input_file_name = arg;
        }

        p_arguments.argv[p_arguments.argc++] = arg;

        arg = shift(&argc, &argv);
    }

    if (p_arguments.input_file_name == NULL) {
        usage(stderr, program_name);
        fprintf(stderr, "error: missing input file\n");
        return 1;
    }

    char *input = NULL;
    int size    = 0;

    if ((size = read_file_content(p_arguments.input_file_name, &input)) < 0) return 1;

    // Lexing
    M_Lexer lexer = m_lexer_create(p_arguments.input_file_name, input, size);
    M_Token *tokens_head = m_lexer_tokenize(&lexer);

    if (lexer.errors) return 1;

    // Parsing
    M_Ast *ast = parse_expression(p_arguments.input_file_name, tokens_head);

    if (ast == NULL) return 0;

    if (ast->errors > 0) {
        m_lexer_free(&lexer);
        ast_free(ast);

        return 1;
    }

    // Interpreting
    M_Interpreter *interpreter = m_interpreter_create(ast, p_arguments.argc, p_arguments.argv);

    m_interpreter_run(interpreter);
    m_interpreter_free(interpreter);
    free(input);

    return 0;
}
