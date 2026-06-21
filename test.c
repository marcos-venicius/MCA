#include <math.h>
#include <string.h>
#include <stdio.h>
#include <assert.h>

#define CLIBS_HT_IMPLEMENTATION
#include "./lexer.h"
#include "./ast.h"
#include "./interpreter.h"
#define CLIBS_ARENA_IMPLEMENTATION
#include "./arena.h"

static long errors = 0;

static inline void LOG_ERROR(const char *expression, M_Value expected, M_Value *actual) {
    errors++;

    fprintf(stderr, "  \033[1;31mFAIL\033[0m '%s'\n", expression);
    if (actual == NULL) {
        switch (expected.type) {
            case M_T_INT:
                fprintf(stderr, "       expected: int(%ld)\n", expected.as.integer);
                break;
            case M_T_FLOAT:
                fprintf(stderr, "       expected: float(%lf)\n", expected.as.floating);
                break;
            default:
                fprintf(stderr, "       expected: broken(%d)\n", expected.type);
                break;
        }
    } else {
        switch (expected.type) {
            case M_T_INT:
                fprintf(stderr, "       expected: int(%ld)", expected.as.integer);
                break;
            case M_T_FLOAT:
                fprintf(stderr, "       expected: float(%lf)", expected.as.floating);
                break;
            default:
                fprintf(stderr, "       expected: broken(%d)\n", expected.type);
                break;
        }

        switch (actual->type) {
            case M_T_INT:
                fprintf(stderr, ", actual: int(%ld)\n", actual->as.integer);
                break;
            case M_T_FLOAT:
                fprintf(stderr, ", actual: float(%lf)\n", actual->as.floating);
                break;
            default:
                fprintf(stderr, ", actual: broken(%d)\n", actual->type);
                break;
        }
    }
}

static inline void LOG_SUCCESS(const char *expression, M_Value result) {
    switch (result.type) {
        case M_T_INT:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37mint(%ld)\033[0m\n", expression, result.as.integer);
            break;
        case M_T_FLOAT:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37mfloat(%lf)\033[0m\n", expression, result.as.floating);
            break;
    }
}

static void RUN_TEST_CASE(const char *expression, M_Value expected) {
    M_Lexer lexer = m_lexer_create(NULL, expression, strlen(expression));
    M_Token *tokens = m_lexer_tokenize(&lexer);

    if (m_lexer_finished_with_errors()) {
        LOG_ERROR(expression, expected, NULL);

        return;
    }

    M_Ast *ast = parse_expression(NULL, tokens);

    M_Interpreter *interpreter = m_interpreter_create(ast);

    FILE *dev_null;

#ifdef _WIN32
    dev_null = fopen("nul", "w");
#else
    dev_null = fopen("/dev/null", "w");
#endif

    if (dev_null != NULL) {
        m_interpreter_set_stdout(interpreter, dev_null);
        m_interpreter_set_stderr(interpreter, dev_null);
    } else {
        fprintf(stderr, "\033[1;34m[RUN_TEST_CASE] Warning: Could not open null device.\033[0m\n");
    }

    M_Value evaluated_expression = m_interpreter_run(interpreter);

    if (evaluated_expression.type == expected.type)
    {
        switch (expected.type) {
            case M_T_INT:
                if (expected.as.integer == evaluated_expression.as.integer) {
                    LOG_SUCCESS(expression, expected);
                    goto clear_test_case;
                }
                break;
            case M_T_FLOAT:
                if (expected.as.floating == evaluated_expression.as.floating || (isnan(expected.as.floating) && isnan(evaluated_expression.as.floating))) {
                    LOG_SUCCESS(expression, expected);
                    goto clear_test_case;
                }
                break;
        }
    }

    LOG_ERROR(expression, expected, &evaluated_expression);

clear_test_case:
    m_lexer_free(&lexer);
    m_interpreter_free(interpreter);
}

static inline void TEST_CASE_LABEL(const char *label) {
    fprintf(stderr, "%s:\n", label);
}

#define T_INT(v) (M_Value){ .type = M_T_INT, .as.integer = v }
#define T_FLOAT(v) (M_Value){ .type = M_T_FLOAT, .as.floating = v }

int main(void) {
    TEST_CASE_LABEL("Basic arithmetic");
    RUN_TEST_CASE("1 + 2", T_INT(3));
    RUN_TEST_CASE("10 - 5", T_INT(5));
    RUN_TEST_CASE("3 * 4", T_INT(12));
    RUN_TEST_CASE("20 / 4", T_INT(5));
    RUN_TEST_CASE("10 % 3", T_INT(1));

    TEST_CASE_LABEL("Operator precedence");
    RUN_TEST_CASE("1 + 2 * 3", T_INT(7));
    RUN_TEST_CASE("(1 + 2) * 3", T_INT(9));
    RUN_TEST_CASE("10 - 4 / 2", T_INT(8));
    RUN_TEST_CASE("(10 - 4) / 2", T_INT(3));

    TEST_CASE_LABEL("Exponentiation (Right-associative)");
    RUN_TEST_CASE("2 ^ 3", T_INT(8));
    RUN_TEST_CASE("2 ^ 3 ^ 2", T_INT(512));
    RUN_TEST_CASE("(2 ^ 3) ^ 2", T_INT(64));

    TEST_CASE_LABEL("Unary operators");
    RUN_TEST_CASE("-5", T_INT(-5));
    RUN_TEST_CASE("-(-5)", T_INT(5));
    RUN_TEST_CASE("4!", T_INT(24));
    RUN_TEST_CASE("-4!", T_INT(-24));
    RUN_TEST_CASE("(-4)!", T_FLOAT(NAN));

    TEST_CASE_LABEL("Combinations");
    RUN_TEST_CASE("2 * 3! + 4 ^ 2 / -2", T_INT(4));
    RUN_TEST_CASE("-((((5.0! + -20) / 10) ^ 2) % 11) * 10 * -1 - 10", T_FLOAT(0));

    TEST_CASE_LABEL("Call builtin functions");
    RUN_TEST_CASE("abs(abs(-1) - 2)", T_INT(1));
    RUN_TEST_CASE("(abs(-1) * 2) ^ 2", T_INT(4));
    RUN_TEST_CASE("abs(min(abs(-1), max(-5, -4)) * 1)", T_INT(4));
    RUN_TEST_CASE("max(abs(-12), 8) * sin(rad(30)) + (16 / 2)", T_FLOAT(14));
    RUN_TEST_CASE("PI()", T_FLOAT(M_PI));
    RUN_TEST_CASE("E()", T_FLOAT(M_E));
    RUN_TEST_CASE("abs(-15.5)", T_FLOAT(15.5));
    RUN_TEST_CASE("max(10.5, 20.0)", T_INT(20));
    RUN_TEST_CASE("min(10.5, 20.0)", T_FLOAT(10.5));
    RUN_TEST_CASE("sin(0)", T_INT(0));
    RUN_TEST_CASE("cos(0)", T_INT(1));
    RUN_TEST_CASE("tan(0)", T_INT(0));
    RUN_TEST_CASE("rad(180)", T_FLOAT(M_PI));
    RUN_TEST_CASE("deg(3.14159265358979323846)", T_INT(180));
    RUN_TEST_CASE("sqrt(25)", T_INT(5));
    RUN_TEST_CASE("log(1)", T_INT(0));
    RUN_TEST_CASE("log10(1000)", T_INT(3));
    RUN_TEST_CASE("exp(1)", T_FLOAT(M_E));
    RUN_TEST_CASE("floor(4.8)", T_INT(4));
    RUN_TEST_CASE("ceil(4.2)", T_INT(5));
    RUN_TEST_CASE("round(4.5)", T_INT(5));
    RUN_TEST_CASE("round(4.4)", T_INT(4));
    RUN_TEST_CASE("type(4.4)", T_FLOAT(4.4));
    RUN_TEST_CASE("type(4)", T_INT(4));

    TEST_CASE_LABEL("Binary operators (equality & relational)");
    RUN_TEST_CASE("5 == 5", T_INT(1));
    RUN_TEST_CASE("10 == 5", T_INT(0));
    RUN_TEST_CASE("0 == 0", T_INT(1));
    RUN_TEST_CASE("10 != 5", T_INT(1));
    RUN_TEST_CASE("5 != 5", T_INT(0));
    RUN_TEST_CASE("0 != 0", T_INT(0));
    RUN_TEST_CASE("10 > 5", T_INT(1));
    RUN_TEST_CASE("5 > 10", T_INT(0));
    RUN_TEST_CASE("5 > 5", T_INT(0));
    RUN_TEST_CASE("5 < 10", T_INT(1));
    RUN_TEST_CASE("10 < 5", T_INT(0));
    RUN_TEST_CASE("5 < 5", T_INT(0));
    RUN_TEST_CASE("10 >= 5", T_INT(1));
    RUN_TEST_CASE("5 >= 5", T_INT(1));
    RUN_TEST_CASE("5 >= 10", T_INT(0));
    RUN_TEST_CASE("5 <= 10", T_INT(1));
    RUN_TEST_CASE("5 <= 5", T_INT(1));
    RUN_TEST_CASE("10 <= 5", T_INT(0));
    RUN_TEST_CASE("1 + 2 == 3", T_INT(1));
    RUN_TEST_CASE("10 - 5 > 2 * 2", T_INT(1));
    RUN_TEST_CASE("5 < 3 + 4", T_INT(1));
    RUN_TEST_CASE("10 == 5 * 2 != 0", T_INT(1));
    RUN_TEST_CASE("0 == 1 < 2", T_INT(0));

    TEST_CASE_LABEL("Printing (return last argument)");
    RUN_TEST_CASE("print()", T_INT(0));
    RUN_TEST_CASE("print(PI())", T_FLOAT(M_PI));
    RUN_TEST_CASE("print(PI(), E(), 10)", T_INT(10));
    RUN_TEST_CASE("println()", T_INT(0));
    RUN_TEST_CASE("println(PI())", T_FLOAT(M_PI));
    RUN_TEST_CASE("println(PI(), E(), 10)", T_INT(10));

    TEST_CASE_LABEL("Global variables");
    RUN_TEST_CASE("x = 10", T_INT(10));
    RUN_TEST_CASE("y = x = 10", T_INT(10));
    RUN_TEST_CASE("y = x = 10;y", T_INT(10));
    RUN_TEST_CASE("x = 10; y = -5.5; z = abs(x * y); println(x, y, z, x + y + z)", T_FLOAT(59.5));

    TEST_CASE_LABEL("Loops");
    RUN_TEST_CASE("n = 10; while n < 20 { n = n + 1 }", T_INT(20));
    RUN_TEST_CASE("a = 0; b = 1; n = 0; while n < 15 { n = n + 1; t = a; a = b; b = t + b; a }", T_INT(610)); // fib
    RUN_TEST_CASE("while 0 {}", T_INT(0));

    TEST_CASE_LABEL("Break");
    RUN_TEST_CASE("r = while 1 { n = 10; break 11.3; println(0); }; r", T_FLOAT(11.3));
    RUN_TEST_CASE("r = while 1 { n = 10; break; println(10); }; r", T_INT(0));
    RUN_TEST_CASE("r = while 1 { n = 10; break floor(10 * 10 - cos(45)); println(10); }; r", T_INT(99));

    TEST_CASE_LABEL("If's");
    RUN_TEST_CASE("x = 10; if x == 10 { x = 11.3 }", T_FLOAT(11.3));
    RUN_TEST_CASE("x = if 10 != 10.1 { 1337 }", T_INT(1337));
    RUN_TEST_CASE("x = if 10 != 10.1 {}", T_INT(0));

    TEST_CASE_LABEL("Elif's");
    RUN_TEST_CASE("if 10 == 10.1 { 1337 } elif 20 == 21 { 1 } elif 20 == 20 { 56 } elif 1 == 1 { } else { 42 }", T_INT(56));
    RUN_TEST_CASE("if 10 == 10.1 { 1337 } elif 20 == 21 { 1 } elif 20 == 22 { 56 } elif 1 == 1 { 33 } else { 42 }", T_INT(33));
    RUN_TEST_CASE("if 10 == 10 { 1337 } elif 20 == 21 { 1 } elif 20 == 22 { 56 } elif 1 == 1 { 33 } else { 42 }", T_INT(1337));
    RUN_TEST_CASE("if 10 == 11 { 1337 } elif 20 == 21 { 1 } elif 20 == 22 { 56 } elif 2 == 1 { 33 } else { 42 }", T_INT(42));

    TEST_CASE_LABEL("Else's");
    RUN_TEST_CASE("if 10 == 10.1 { 1337 } else { 42 }", T_INT(42));
    RUN_TEST_CASE("if 10 == 10.1 { 1337 } else { }", T_INT(0));

    TEST_CASE_LABEL("Logical operators");
    RUN_TEST_CASE("if 10 == 10 and 20 == 20 { 20 } else { }", T_INT(20));
    RUN_TEST_CASE("if 10 == 11 and 20 == 20 { 20 } else { 10 }", T_INT(10));
    RUN_TEST_CASE("if 10 == 11 or 20 == 20 { 20 } else { 10 }", T_INT(20));
    RUN_TEST_CASE("if 10 == 11 or 20 == 21 { 20 } else { 10 }", T_INT(10));

    TEST_CASE_LABEL("Not operator");
    RUN_TEST_CASE("if !(10 == 11 or 20 == 21) { 20 } else { 10 }", T_INT(20));
    RUN_TEST_CASE("!(1 == 0)", T_INT(1));
    RUN_TEST_CASE("!1.4", T_INT(0));

    if (errors == 1) {
        fprintf(stderr, "\n\033[1;31m1\033[0m test case broke\n");
    } else if (errors > 1) {
        fprintf(stderr, "\n\033[1;31m%ld\033[0m test cases broke\n", errors);
    } else {
        fprintf(stderr, "\nAll tests \033[1;32mpass successfully\033[0m\n");
    }
}
