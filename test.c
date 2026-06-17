#include <math.h>
#include <string.h>
#include <stdio.h>
#include <assert.h>

#include "./lexer.h"
#include "./ast.h"
#include "./evaluator.h"
#define CLIBS_ARENA_IMPLEMENTATION
#include "./arena.h"


static inline void LOG_ERROR(const char *expression, double expected, double *actual) {
    fprintf(stderr, "  \033[1;31mFAIL\033[0m '%s'\n", expression);
    if (actual == NULL) {
        fprintf(stderr, "       expected: %f\n", expected);
    } else {
        fprintf(stderr, "       expected: %f, actual: %f\n", expected, *actual);
    }
}

static inline void LOG_SUCCESS(const char *expression, double result) {
    fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37m%f\033[0m\n", expression, result);
}

static void RUN_TEST_CASE(const char *expression, double expected) {
    M_Lexer lexer = m_lexer_create(NULL, expression, strlen(expression));
    M_Token *tokens = m_lexer_tokenize(&lexer);

    if (m_lexer_finished_with_errors()) {
        LOG_ERROR(expression, expected, NULL);

        return;
    }

    M_Ast *ast = parse_expression(NULL, tokens);

    assert(ast->expressions_array_length == 1 && "RUN_TEST_CASE: we do not handle multiple (or empty) expressions in this test case scenario");

    double evaluated_expression = evaluate_expression(ast->expressions_array[0]);

    m_lexer_free(&lexer);
    ast_free(ast);

    if (evaluated_expression == expected || (isnan(evaluated_expression) && isnan(expected))) LOG_SUCCESS(expression, expected);
    else LOG_ERROR(expression, expected, &evaluated_expression);
}

static inline void TEST_CASE_LABEL(const char *label) {
    fprintf(stderr, "%s:\n", label);
}

int main(void) {
    TEST_CASE_LABEL("Basic arithmetic");
    RUN_TEST_CASE("1 + 2", 3.0);
    RUN_TEST_CASE("10 - 5", 5.0);
    RUN_TEST_CASE("3 * 4", 12.0);
    RUN_TEST_CASE("20 / 4", 5.0);
    RUN_TEST_CASE("10 % 3", 1.0);

    TEST_CASE_LABEL("Operator precedence");
    RUN_TEST_CASE("1 + 2 * 3", 7.0);
    RUN_TEST_CASE("(1 + 2) * 3", 9.0);
    RUN_TEST_CASE("10 - 4 / 2", 8.0);
    RUN_TEST_CASE("(10 - 4) / 2", 3.0);

    TEST_CASE_LABEL("Exponentiation (Right-associative)");
    RUN_TEST_CASE("2 ^ 3", 8.0);
    RUN_TEST_CASE("2 ^ 3 ^ 2", 512.0);
    RUN_TEST_CASE("(2 ^ 3) ^ 2", 64.0);

    TEST_CASE_LABEL("Unary operators");
    RUN_TEST_CASE("-5", -5.0);
    RUN_TEST_CASE("-(-5)", 5.0);
    RUN_TEST_CASE("4!", 24.0);
    RUN_TEST_CASE("-4!", -24.0);
    RUN_TEST_CASE("(-4)!", NAN);

    TEST_CASE_LABEL("Combinations");
    RUN_TEST_CASE("2 * 3! + 4 ^ 2 / -2", 4.0);
    RUN_TEST_CASE("-((((5.0! + -20) / 10) ^ 2) % 11) * 10 * -1 - 10", 0.0);

    TEST_CASE_LABEL("Call builtin functions");
    RUN_TEST_CASE("abs(-1)", 1);
    RUN_TEST_CASE("abs(abs(-1) - 2)", 1);
    RUN_TEST_CASE("(abs(-1) * 2) ^ 2", 4);
    RUN_TEST_CASE("min(-10.5, -10.4)", -10.5);
    RUN_TEST_CASE("max(-10.5, -10.4)", -10.4);
    RUN_TEST_CASE("abs(min(abs(-1), max(-5, -4)) * 1)", 4);
    RUN_TEST_CASE("rad(180)", M_PI);
    RUN_TEST_CASE("deg(rad(180))", 180);
    RUN_TEST_CASE("sin(30)", sin(30));
    RUN_TEST_CASE("sin(rad(30))", sin(30.0 * (M_PI / 180.0)));
    RUN_TEST_CASE("cos(0)", 1);
    RUN_TEST_CASE("tan(23)", tan(23));
    RUN_TEST_CASE("sqrt(25)", 5);
    RUN_TEST_CASE("sqrt(45)", sqrt(45));
    RUN_TEST_CASE("log(1024)", log(1024));
    RUN_TEST_CASE("log10(100)", 2);
    RUN_TEST_CASE("exp(2)", exp(2));
    RUN_TEST_CASE("floor(rad(180))", floor(M_PI));
    RUN_TEST_CASE("ceil(rad(180))", ceil(M_PI));
    RUN_TEST_CASE("round(rad(180))", round(M_PI));
    RUN_TEST_CASE("pi()", M_PI);
    RUN_TEST_CASE("e()", M_E);
}
