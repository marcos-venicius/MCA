#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include <stdlib.h>
#include <math.h>
#include <assert.h>
#include "./lexer.h"
#include "./ast.h"
#include "./log.h"
#include "./io.h"

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

    if (expr->kind == M_EK_NUMBER) {
        printf("%f", expr->number);
    } else if (expr->kind == M_EK_UNARY) {
        if (expr->unary.op == M_UNARY_MINUS_OP) {
            printf("-(");
            print_expr(expr->unary.operand);
            printf(")");
        } else if (expr->unary.op == M_UNARY_FACTORIAL_OP) {
            printf("(");
            print_expr(expr->unary.operand);
            printf(")!");
        }
    } else {
        printf("(");
        print_expr(expr->binary.left);
        switch (expr->binary.op) {
            case M_BINARY_PLUS_OP: printf(" + "); break;
            case M_BINARY_TIMES_OP: printf(" * "); break;
            case M_BINARY_DIVIDE_OP: printf(" / "); break;
            case M_BINARY_SUBTRACT_OP: printf(" - "); break;
            case M_BINARY_MOD_OP: printf(" %% "); break;
            case M_BINARY_POW_OP: printf(" ^ "); break;
        }
        print_expr(expr->binary.right);
        printf(")");
    }
}

double calculate_factorial(double number) {
    if (number < 0 && number == (int)number) return NAN;
    
    return tgamma(number + 1.0);
}

double evaluate_expression(M_Expression *expression) {
    if (expression->kind == M_EK_NUMBER) return expression->number;

    if (expression->kind == M_EK_UNARY) {
        switch (expression->unary.op) {
            case M_UNARY_MINUS_OP: return -evaluate_expression(expression->unary.operand);
            case M_UNARY_FACTORIAL_OP: return calculate_factorial(evaluate_expression(expression->unary.operand));
            default:
                assert(0 && "evaluate_expression: invalid unary expression operator");
        }
    }

    switch (expression->binary.op) {
        case M_BINARY_PLUS_OP: return evaluate_expression(expression->binary.left) + evaluate_expression(expression->binary.right);
        case M_BINARY_TIMES_OP: return evaluate_expression(expression->binary.left) * evaluate_expression(expression->binary.right);
        case M_BINARY_DIVIDE_OP: return evaluate_expression(expression->binary.left) / evaluate_expression(expression->binary.right);
        case M_BINARY_SUBTRACT_OP: return evaluate_expression(expression->binary.left) - evaluate_expression(expression->binary.right);
        case M_BINARY_MOD_OP: return fmod(evaluate_expression(expression->binary.left), evaluate_expression(expression->binary.right));
        case M_BINARY_POW_OP: return pow(evaluate_expression(expression->binary.left), evaluate_expression(expression->binary.right));
        default:
            assert(0 && "evaluate_expression: invalid binary expression operator");
    }
}

int compile_math(const char *filename, const char *string, const size_t string_size, M_Expression **expression_output) {
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
    }

    if (expression_output == NULL) {
        m_lexer_free(&lexer);
        return 0;
    }

    *expression_output = parse_expression(&tokens);

    if (*expression_output == NULL) {
        m_lexer_free(&lexer);
        LOG("[*] There is no expression\n");
        return 0;
    }

    if (is_log_enabled()) {
        print_expr(*expression_output);
        printf("\n");
    }

    m_lexer_free(&lexer);

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

    int result;
    M_Expression *expression;

    if (p_arguments.input_file_name != NULL) {
        char *input;
        int size;

        if ((size = read_file_content(p_arguments.input_file_name, &input)) < 0) return 1;

        result = compile_math(p_arguments.input_file_name, input, size, &expression);
        free(input);

    } else {
        result = compile_math(NULL, p_arguments.math, strlen(p_arguments.math), &expression);
    }

    if (result == 0) {
        double evaluated_expression = evaluate_expression(expression);

        printf("%f\n", evaluated_expression);
    }

    return result;
}
