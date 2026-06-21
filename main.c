#define CLIBS_HT_IMPLEMENTATION

#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include <stdlib.h>
#include <assert.h>
#include "./lexer.h"
#include "./ast.h"
#include "./log.h"
#include "./io.h"
#include "./interpreter.h"


#define CLIBS_ARENA_IMPLEMENTATION
#include "./arena.h"

typedef struct {
    const char *input_file_name;
    const char *math;
} ProgramArguments;

void usage(FILE *stream, const char *program_name) {
    fprintf(stream, "USAGE: %s [math] [flags]\n\n", program_name);
    fprintf(stream, "    -i   [file]         evaluate math inside a file\n");
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

void print_expr(M_Expression *expr) {
    if (expr == NULL) return;

    if (expr->kind == M_EK_INT) {
        printf("%ld", expr->integer);
    } else if (expr->kind == M_EK_FLOAT) {
        printf("%lf", expr->floating);
    } else if (expr->kind == M_EK_UNARY) {
        if (expr->unary.op == M_UNARY_MINUS_OP) {
            printf("-(");
            print_expr(expr->unary.operand);
            printf(")");
        } else if (expr->unary.op == M_UNARY_FACTORIAL_OP) {
            printf("(");
            print_expr(expr->unary.operand);
            printf(")!");
        } else if (expr->unary.op == M_UNARY_NOT_OP) {
            printf("!");
            print_expr(expr->unary.operand);
        }
    } else if (expr->kind == M_EK_CALL) {
        printf("%.*s(", expr->call.fn_name_length, expr->call.fn_name);
        for (int i = 0; i < expr->call.arguments_length; i++) {
            if (i > 0) printf(", ");

            print_expr(expr->call.arguments[i]);
        }
        printf(")");
    } else if (expr->kind == M_EK_BINARY) {
        printf("(");
        print_expr(expr->binary.left);
        switch (expr->binary.op) {
            case M_BINARY_PLUS_OP: printf(" + "); break;
            case M_BINARY_TIMES_OP: printf(" * "); break;
            case M_BINARY_DIVIDE_OP: printf(" / "); break;
            case M_BINARY_SUBTRACT_OP: printf(" - "); break;
            case M_BINARY_MOD_OP: printf(" %% "); break;
            case M_BINARY_POW_OP: printf(" ^ "); break;

            case M_BINARY_AND_OP: printf(" and "); break;
            case M_BINARY_OR_OP: printf(" or "); break;

            case M_BINARY_EQUAL_OP: printf(" == "); break;
            case M_BINARY_NOT_EQUAL_OP: printf(" != "); break;
            case M_BINARY_GT_OP: printf(" > "); break;
            case M_BINARY_LT_OP: printf(" < "); break;
            case M_BINARY_GTE_OP: printf(" >= "); break;
            case M_BINARY_LTE_OP: printf(" <= "); break;
        }
        print_expr(expr->binary.right);
        printf(")");
    } else {
        assert(0 && "print_expr: missing M_Expression_Kind handler");
    }
}

int compile(const char *filename, const char *string, const size_t string_size, M_Ast **ast_output) {
    LOG("[*] compiling math\n");

    M_Lexer lexer = m_lexer_create(filename, string, string_size);

    M_Token *tokens = m_lexer_tokenize(&lexer);

    if (m_lexer_finished_with_errors()) {
        return -1;
    }

    if (tokens == NULL) {
        LOG("[*] There is no tokens\n");
        return 0;
    }

    if (is_log_enabled()) {
        printf("TOKENS: \n");
        for (M_Token *token = tokens; token != NULL; token = token->next) {
            printf("    <Token value=[%.*s] kind=[%s] />\n", (int)token->size, token->value, m_lexer_token_kind_display_name(token->kind));
        }
        printf("\n");
    }

    if (ast_output == NULL) {
        m_lexer_free(&lexer);
        return 0;
    }

    *ast_output = parse_expression(filename, tokens);

    if (*ast_output == NULL) {
        m_lexer_free(&lexer);
        LOG("[*] There is no expression\n");
        return 0;
    }

    if (is_log_enabled()) {
        for (int i = 0; i < (*ast_output)->expressions_array_length; i++) {
            printf("EXP %d:\n", i + 1);
            print_expr((*ast_output)->expressions_array[i]);
            printf("\n");
        }
    }

    return 0;
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
    init_logging();

    const char *program_name = shift(&argc, &argv);

    ProgramArguments p_arguments = {0};

    const char *arg = shift(&argc, &argv);

    while (arg != NULL) {
        if (cmp_arg(arg, "-h")) {
            usage(stdout, program_name);
            return 0;
        } else if (cmp_arg(arg, "-i")) {
            const char *value = shift(&argc, &argv);

            if (value == NULL) {
                usage(stderr, program_name);
                fprintf(stderr, "error: missing value for flag -i\n");
                return 1;
            }

            p_arguments.input_file_name = value;
        } else {
            p_arguments.math = arg;
        }

        arg = shift(&argc, &argv);
    }

    if (p_arguments.input_file_name == NULL && p_arguments.math == NULL) {
        usage(stderr, program_name);
        fprintf(stderr, "error: please, provide some math or -i flag\n");
        return 1;
    }

    int result = 0;
    M_Ast *ast = NULL;
    char *input = NULL;

    if (p_arguments.input_file_name != NULL) {
        int size;

        if ((size = read_file_content(p_arguments.input_file_name, &input)) < 0) return 1;

        result = compile(p_arguments.input_file_name, input, size, &ast);
    } else {
        result = compile(NULL, p_arguments.math, strlen(p_arguments.math), &ast);
    }

    if (result != 0) {
        if (ast != NULL) ast_free(ast);

        return result;
    }

    if (ast == NULL) return 0;

    M_Interpreter *interpreter = m_interpreter_create(ast);

    m_interpreter_run(interpreter);
    m_interpreter_free(interpreter);

    if (input != NULL) free(input);

    return 0;
}
