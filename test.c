#include <math.h>
#include <string.h>
#include <stdio.h>
#include <assert.h>

#include "./lexer.h"
#include "./ast.h"
#include "./interpreter.h"
#include "./arena.h"
#include "./builtins/map.h"

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

#ifndef M_E
#define M_E 2.7182818284590452354
#endif

static long success = 0;
static long errors = 0;
static long tests_count = 0;
static int m_argc;
static const char *m_argv[] = {"fakename.mca", "fakearg"};

static void LOG_ERROR(const char *expression, M_Value expected, M_Value *actual, const char *file, int line) {
    errors++;

    fprintf(stderr, "  \033[1;31mFAIL\033[0m '%s' \033[0;33m(%s:%d)\033[0m\n", expression, file, line);

    static_assert(M_T_COUNT == 257, "LOG_ERROR: missing M_Value_Type handler");
    switch (expected.type) {
        case M_T_INT:
            fprintf(stderr, "       expected: int(%ld)", expected.as.integer);
            break;
        case M_T_FLOAT:
            fprintf(stderr, "       expected: float(%lf)", expected.as.floating);
            break;
        case M_T_BOOL:
            fprintf(stderr, "       expected: bool(%s)", expected.as.boolean ? "true" : "false");
            break;
        case M_T_UNIT:
            fprintf(stderr, "       expected: unit");
            break;
        case M_T_STRING:
            fprintf(stderr, "       expected: string(\"%.*s\")", expected.as.string.value_length, expected.as.string.value);
            break;
        case M_T_ARRAY:
            fprintf(stderr, "       expected: array(%d)", expected.as.array->length);
            break;
        case M_T_MAP:
            fprintf(stderr, "       expected: map(%d)", expected.as.map->size);
            break;
        case M_T_MAP_IT:
            fprintf(stderr, "       expected: iter<map(%d)>", expected.as.map_it->map->size);
            break;
        case M_T_FN:
            fprintf(stderr, "       expected: fn(...%d)", expected.as.fn->Fn.arguments_length);
            break;
        default:
            fprintf(stderr, "       expected: broken(%d)", expected.type);
            break;
    }

    if (actual != NULL) {
        static_assert(M_T_COUNT == 257, "LOG_ERROR: missing M_Value_Type handler");
        switch (actual->type) {
            case M_T_INT:
                fprintf(stderr, ", actual: int(%ld)", actual->as.integer);
                break;
            case M_T_FLOAT:
                fprintf(stderr, ", actual: float(%lf)", actual->as.floating);
                break;
            case M_T_BOOL:
                fprintf(stderr, ", actual: bool(%s)", actual->as.boolean ? "true" : "false");
                break;
            case M_T_UNIT:
                fprintf(stderr, ", actual: unit");
                break;
            case M_T_STRING:
                fprintf(stderr, ", actual: string(\"%.*s\")", actual->as.string.value_length, actual->as.string.value);
                break;
            case M_T_MAP:
                fprintf(stderr, ", actual: map(%d)", actual->as.map->size);
                break;
            case M_T_MAP_IT:
                fprintf(stderr, ", actual: iter<map(%d)>", actual->as.map_it->map->size);
                break;
            case M_T_FN:
                fprintf(stderr, ", actual: fn(...%d)", actual->as.fn->Fn.arguments_length);
                break;
            default:
                fprintf(stderr, ", actual: broken(%d)", actual->type);
                break;
        }
    }

    fprintf(stderr, "\n");
}

static void LOG_SUCCESS(const char *expression, M_Value result) {
    success++;

    switch (result.type) {
        case M_T_ARRAY:
        case M_T_INT:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37mint(%ld)\033[0m\n", expression, result.as.integer);
            break;
        case M_T_FLOAT:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37mfloat(%lf)\033[0m\n", expression, result.as.floating);
            break;
        case M_T_BOOL:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37mbool(%s)\033[0m\n", expression, result.as.boolean ? "true" : "false");
            break;
        case M_T_UNIT:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37munit\033[0m\n", expression);
            break;
        case M_T_STRING:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37mstring(\"%.*s\")\033[0m\n", expression, result.as.string.value_length, result.as.string.value);
            break;
        case M_T_MAP:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37mmap(%d)\033[0m\n", expression, result.as.map->size);
            break;
        case M_T_MAP_IT:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37miter<map(%d)>\033[0m\n", expression, result.as.map_it->map->size);
            break;
        case M_T_FN:
            fprintf(stderr, "  \033[1;32mPASS\033[0m '%s' => \033[1;37mfn(...%d)\033[0m\n", expression, result.as.fn->Fn.arguments_length);
            break;
        case M_T_COUNT:
            assert(0 && "LOG_SUCCESS: unreachable M_T_COUNT");
            break;
    }
}

static void RUN_TEST_CASE(const char *expression, M_Value expected, const char *file, int line, bool ignore) {
    if (ignore) return;

    tests_count++;

    M_Lexer lexer = m_lexer_create(NULL, expression, strlen(expression));
    M_Token *tokens = m_lexer_tokenize(&lexer);

    if (m_lexer_finished_with_errors()) {
        LOG_ERROR(expression, expected, NULL, file, line);

        m_argc = 0;
        return;
    }

    M_Ast *ast = parse_expression(NULL, tokens);

    M_Interpreter *interpreter = m_interpreter_create(ast, m_argc, m_argv);

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
            case M_T_BOOL:
                if (expected.as.boolean == evaluated_expression.as.boolean) {
                    LOG_SUCCESS(expression, expected);
                    goto clear_test_case;
                }
                break;
            case M_T_UNIT:
                LOG_SUCCESS(expression, expected);
                goto clear_test_case;
            case M_T_STRING:
                if (expected.as.string.value_length == evaluated_expression.as.string.value_length &&
                    strncmp(expected.as.string.value, evaluated_expression.as.string.value, evaluated_expression.as.string.value_length) == 0) {
                    LOG_SUCCESS(expression, expected);
                    goto clear_test_case;
                }
                break;
            case M_T_ARRAY:
                assert(0 && "TODO: implement test case for array");
                break;
            case M_T_FN:
                assert(0 && "TODO: implement test case for functions");
                break;
            case M_T_MAP_IT:
                assert(0 && "TODO: implement test case for map iterators");
                break;
            case M_T_MAP:
                assert(0 && "TODO: implement test case for maps");
                break;
            case M_T_COUNT:
                assert(0 && "RUN_TEST_CASE: unreachable M_T_COUNT");
                break;
        }
    }

    LOG_ERROR(expression, expected, &evaluated_expression, file, line);

clear_test_case:
    m_lexer_free(&lexer);
    m_interpreter_free(interpreter);
    m_argc = 0;
}

static inline void TEST_CASE_LABEL(const char *label) {
    fprintf(stderr, "%s:\n", label);
}

#define T_UNIT() (M_Value){ .type = M_T_UNIT }
#define T_INT(v) (M_Value){ .type = M_T_INT, .as.integer = v }
#define T_FLOAT(v) (M_Value){ .type = M_T_FLOAT, .as.floating = v }
#define T_BOOL(v) (M_Value){ .type = M_T_BOOL, .as.boolean = v }
#define T_STRING(v) (M_Value){ .type = M_T_STRING, .as.string.value = v, .as.string.value_length = strlen(v) }

#if 1
#define TEST_CASE(expr, expected) RUN_TEST_CASE(expr, expected, __FILE__, __LINE__, false)
#else
#define TEST_CASE(expr, expected) RUN_TEST_CASE(expr, expected, __FILE__, __LINE__, true)
#define TEST_CASE_SINGLE(expr, expected) RUN_TEST_CASE(expr, expected, __FILE__, __LINE__, false)
#endif

int main(void) {
    // [[tests]]
    TEST_CASE_LABEL("TOP LEVEL");
    TEST_CASE("", T_UNIT());
    TEST_CASE(";", T_UNIT());

    TEST_CASE_LABEL("Basic arithmetic");
    TEST_CASE("1 + 2", T_INT(3));
    TEST_CASE("10 - 5", T_INT(5));
    TEST_CASE("3 * 4", T_INT(12));
    TEST_CASE("20 / 4", T_INT(5));
    TEST_CASE("10 % 3", T_INT(1));

    TEST_CASE_LABEL("Operator precedence");
    TEST_CASE("1 + 2 * 3", T_INT(7));
    TEST_CASE("(1 + 2) * 3", T_INT(9));
    TEST_CASE("10 - 4 / 2", T_INT(8));
    TEST_CASE("(10 - 4) / 2", T_INT(3));

    TEST_CASE_LABEL("Exponentiation (Right-associative)");
    TEST_CASE("2 ^ 3", T_INT(8));
    TEST_CASE("2 ^ 3 ^ 2", T_INT(512));
    TEST_CASE("(2 ^ 3) ^ 2", T_INT(64));

    TEST_CASE_LABEL("Unary operators");
    TEST_CASE("-5", T_INT(-5));
    TEST_CASE("-(-5)", T_INT(5));
    TEST_CASE("4!", T_INT(24));
    TEST_CASE("-4!", T_INT(-24));
    TEST_CASE("(-4)!", T_FLOAT(NAN));

    TEST_CASE_LABEL("Combinations");
    TEST_CASE("2 * 3! + 4 ^ 2 / -2", T_INT(4));
    TEST_CASE("-((((5.0! + -20) / 10) ^ 2) % 11) * 10 * -1 - 10", T_FLOAT(0));

    TEST_CASE_LABEL("Call builtin functions");
    TEST_CASE("abs(abs(-1) - 2)", T_INT(1));
    TEST_CASE("(abs(-1) * 2) ^ 2", T_INT(4));
    TEST_CASE("abs(min(abs(-1), max(-5, -4)) * 1)", T_INT(4));
    TEST_CASE("max(abs(-12), 8) * sin(rad(30)) + (16 / 2)", T_FLOAT(14));
    TEST_CASE("PI()", T_FLOAT(M_PI));
    TEST_CASE("E()", T_FLOAT(M_E));
    TEST_CASE("abs(-15.5)", T_FLOAT(15.5));
    TEST_CASE("max(10.5, 20.0)", T_FLOAT(20.0));
    TEST_CASE("min(10.5, 20.0)", T_FLOAT(10.5));
    TEST_CASE("sin(0)", T_INT(0));
    TEST_CASE("deg(asin(1))", T_INT(90));
    TEST_CASE("deg(acos(0))", T_INT(90));
    TEST_CASE("cos(0)", T_INT(1));
    TEST_CASE("tan(0)", T_INT(0));
    TEST_CASE("rad(180)", T_FLOAT(M_PI));
    TEST_CASE("deg(3.14159265358979323846)", T_INT(180));
    TEST_CASE("sqrt(25)", T_INT(5));
    TEST_CASE("log(1)", T_INT(0));
    TEST_CASE("log10(1000)", T_INT(3));
    TEST_CASE("exp(1)", T_FLOAT(M_E));
    TEST_CASE("floor(4.8)", T_INT(4));
    TEST_CASE("ceil(4.2)", T_INT(5));
    TEST_CASE("round(4.5)", T_INT(5));
    TEST_CASE("round(4.4)", T_INT(4));
    TEST_CASE("type(4.4)", T_STRING("float"));
    TEST_CASE("type(4)", T_STRING("int"));
    TEST_CASE("srand(4)", T_UNIT());
    TEST_CASE("rand(1, 10)", T_INT(2));
    TEST_CASE("len('Hello World')", T_INT(11));
    TEST_CASE("argc()", T_INT(0));
    m_argc = 1;
    TEST_CASE("argc()", T_INT(1));
    m_argc = 1;
    TEST_CASE("argv(0)", T_STRING("fakename.mca"));
    m_argc = 2;
    TEST_CASE("argv(1)", T_STRING("fakearg"));
    TEST_CASE("is_int(1)", T_BOOL(true));
    TEST_CASE("is_int(1.3)", T_BOOL(false));
    TEST_CASE("is_float(1)", T_BOOL(false));
    TEST_CASE("is_float(1.3)", T_BOOL(true));
    TEST_CASE("is_string(1)", T_BOOL(false));
    TEST_CASE("is_string('1.3')", T_BOOL(true));
    TEST_CASE("is_bool(1)", T_BOOL(false));
    TEST_CASE("is_bool(false)", T_BOOL(true));
    TEST_CASE("is_unit(1)", T_BOOL(false));
    TEST_CASE("is_unit(?)", T_BOOL(true));
    TEST_CASE("'Hello, World'[7]", T_STRING("W"));
    TEST_CASE("read_entire_file('./test/file.txt')", T_STRING("Hello World\n"));
    TEST_CASE("select('Hello, World', 7, 12)", T_STRING("World"));
    TEST_CASE("select('heyhey', 0, 6)", T_STRING("heyhey"));
    TEST_CASE("select('heyhey', 2, 3)", T_STRING("y"));
    TEST_CASE("select('heyhey', 3, 6)", T_STRING("hey"));
    TEST_CASE("ord('a')", T_INT('a'));
    TEST_CASE("ord('b')", T_INT('b'));
    TEST_CASE("ord('z')", T_INT('z'));
    TEST_CASE("format('Hello ', 'World!', ' I am ', 5, ' years old. And ', 5.6, ' feet. I am a ', true, ' tall. I am not ', false)", T_STRING("Hello World! I am 5 years old. And 5.6 feet. I am a true tall. I am not false"));

    TEST_CASE_LABEL("Binary operators (equality & relational)");
    TEST_CASE("5 == 5", T_BOOL(true));
    TEST_CASE("10 == 5", T_BOOL(false));
    TEST_CASE("0 == 0", T_BOOL(true));
    TEST_CASE("10 != 5", T_BOOL(true));
    TEST_CASE("5 != 5", T_BOOL(false));
    TEST_CASE("0 != 0", T_BOOL(false));
    TEST_CASE("10 > 5", T_BOOL(true));
    TEST_CASE("5 > 10", T_BOOL(false));
    TEST_CASE("5 > 5", T_BOOL(false));
    TEST_CASE("5 < 10", T_BOOL(true));
    TEST_CASE("10 < 5", T_BOOL(false));
    TEST_CASE("5 < 5", T_BOOL(false));
    TEST_CASE("10 >= 5", T_BOOL(true));
    TEST_CASE("5 >= 5", T_BOOL(true));
    TEST_CASE("5 >= 10", T_BOOL(false));
    TEST_CASE("5 <= 10", T_BOOL(true));
    TEST_CASE("5 <= 5", T_BOOL(true));
    TEST_CASE("10 <= 5", T_BOOL(false));
    TEST_CASE("1 + 2 == 3", T_BOOL(true));
    TEST_CASE("10 - 5 > 2 * 2", T_BOOL(true));
    TEST_CASE("5 < 3 + 4", T_BOOL(true));
    TEST_CASE("10 == 5 * 2 != 0", T_BOOL(true));
    TEST_CASE("0 == 1 < 2", T_BOOL(false));
    TEST_CASE("'Hello' == 'hello'", T_BOOL(false));
    TEST_CASE("'Hello' == 'Hello'", T_BOOL(true));
    TEST_CASE("'Hello World' == 'Hello'", T_BOOL(false));
    TEST_CASE("'Hello World' != 'Hello'", T_BOOL(true));
    TEST_CASE("'Hello' != 'Hello'", T_BOOL(false));

    TEST_CASE_LABEL("Printing (return last argument)");
    TEST_CASE("print()", T_UNIT());
    TEST_CASE("print(PI())", T_FLOAT(M_PI));
    TEST_CASE("print(PI(), E(), 10)", T_INT(10));
    TEST_CASE("println()", T_UNIT());
    TEST_CASE("println(PI())", T_FLOAT(M_PI));
    TEST_CASE("println(PI(), E(), 10)", T_INT(10));

    TEST_CASE_LABEL("Global variables");
    TEST_CASE("x = 10", T_INT(10));
    TEST_CASE("y = x = 10", T_INT(10));
    TEST_CASE("y = x = 10;y", T_INT(10));
    TEST_CASE("x = 10; y = -5.5; z = abs(x * y); println(x, y, z, x + y + z)", T_FLOAT(59.5));

    TEST_CASE_LABEL("Assignment");
    TEST_CASE("y = x = 10; x + y", T_INT(20));
    TEST_CASE("i = 0; while i < 10 { i += 1 }", T_INT(10));
    TEST_CASE("i = 10; while i > 10 { i -= 1 }", T_UNIT());
    TEST_CASE("i = 10; i += 2", T_INT(12));
    TEST_CASE("i = 10; i += 2; i", T_INT(12));
    TEST_CASE("i = 10; i -= 2", T_INT(8));
    TEST_CASE("i = 10; i -= 2; i", T_INT(8));

    TEST_CASE_LABEL("Loops");
    TEST_CASE("n = 10; while n < 20 { n = n + 1 }", T_INT(20));
    TEST_CASE("a = 0; b = 1; n = 0; while n < 15 { n = n + 1; t = a; a = b; b = t + b; a }", T_INT(610)); // fib
    TEST_CASE("while false {}", T_UNIT());
    TEST_CASE("while false;", T_UNIT());
    TEST_CASE("n = 0; while n < 10  n += 1", T_INT(10));
    TEST_CASE("i = 0; n = 0; while n < 10  n += 1  i += 1", T_INT(1)); // when while does not have a block it accept a single expression as body

    TEST_CASE_LABEL("Break");
    TEST_CASE("r = while 1 { n = 10; break 11.3; println(0); }; r", T_FLOAT(11.3));
    TEST_CASE("r = while 1 { n = 10; break; println(10); }; r", T_UNIT()); // break expressions without values returns UNIT
    TEST_CASE("r = while 1 { n = 10; break floor(10 * 10 - cos(45)); println(10); }; r", T_INT(99));

    TEST_CASE_LABEL("If's");
    TEST_CASE("x = 10; if x == 10 { x = 11.3 }", T_FLOAT(11.3));
    TEST_CASE("if 0 == 0;", T_UNIT()); // empty if expression, equivalent to if 0 == 0 {}
    TEST_CASE("x = if 10 != 10.1 { 1337 }", T_INT(1337));
    TEST_CASE("x = if 10 != 10.1 {}", T_UNIT());
    TEST_CASE("if false 0 elif false 1 else true 2", T_INT(2));
    TEST_CASE("if false { 0 } elif false { 1 } else true 2", T_INT(2));
    TEST_CASE("if false { 0 } elif false { 1 } else { 2; 3; 4; }", T_INT(4));
    TEST_CASE("srand(4); a = if rand(0, 10) % 2 == 0 'Ok' else 'Fail'; println(a)", T_STRING("Ok"));

    TEST_CASE_LABEL("Elif's");
    TEST_CASE("if 0 == 1; elif 0 == 0;", T_UNIT()); // empty elif expression, equivalent to elif 0 == 0 {}
    TEST_CASE("if 10 == 10.1 { 1337 } elif 20 == 21 { 1 } elif 20 == 20 { 56 } elif 1 == 1 { } else { 42 }", T_INT(56));
    TEST_CASE("if 10 == 10.1 { 1337 } elif 20 == 21 { 1 } elif 20 == 22 { 56 } elif 1 == 1 { 33 } else { 42 }", T_INT(33));
    TEST_CASE("if 10 == 10 { 1337 } elif 20 == 21 { 1 } elif 20 == 22 { 56 } elif 1 == 1 { 33 } else { 42 }", T_INT(1337));
    TEST_CASE("if 10 == 11 { 1337 } elif 20 == 21 { 1 } elif 20 == 22 { 56 } elif 2 == 1 { 33 } else { 42 }", T_INT(42));
    TEST_CASE("if false 0 elif true { ;;;; } else true", T_UNIT());

    TEST_CASE_LABEL("Else's");
    TEST_CASE("if 0 == 1; elif 0 == 1; else;", T_UNIT()); // empty else expression, equivalent to else {}
    TEST_CASE("if 10 == 10.1 { 1337 } else { 42 }", T_INT(42));
    TEST_CASE("if 10 == 10.1 { 1337 } else { }", T_UNIT());
    TEST_CASE("if 10 == 10.1 { 1337 } else { ;; }", T_UNIT());

    TEST_CASE_LABEL("Logical operators");
    TEST_CASE("if 10 == 10 and 20 == 20 { 20 } else { }", T_INT(20));
    TEST_CASE("if 10 == 11 and 20 == 20 { 20 } else { 10 }", T_INT(10));
    TEST_CASE("if 10 == 11 or 20 == 20 { 20 } else { 10 }", T_INT(20));
    TEST_CASE("if 10 == 11 or 20 == 21 { 20 } else { 10 }", T_INT(10));
    TEST_CASE("if 10 == 10 or n == x { 20 } else { 0 }", T_INT(20)); // should work because the right side is lazy-evaluated
    TEST_CASE("if 10 == 11 and n == x { 0 } else { 20 }", T_INT(20)); // should work because the right side is lazy-evaluated

    TEST_CASE_LABEL("Not operator");
    TEST_CASE("if !(10 == 11 or 20 == 21) { 20 } else { 10 }", T_INT(20));
    TEST_CASE("!(1 == 0)", T_BOOL(true));
    TEST_CASE("!1.4", T_BOOL(false));

    TEST_CASE_LABEL("Booleans");
    TEST_CASE("true", T_BOOL(true));
    TEST_CASE("false", T_BOOL(false));
    TEST_CASE("!true", T_BOOL(false));
    TEST_CASE("!false", T_BOOL(true));

    TEST_CASE_LABEL("Type casting");
    TEST_CASE("as_int(10.5)", T_INT(10));
    TEST_CASE("as_int(true)", T_INT(1));
    TEST_CASE("as_int('-103956')", T_INT(-103956));
    TEST_CASE("as_int('103956')", T_INT(103956));
    TEST_CASE("as_int('103956'[2])", T_INT(3));
    TEST_CASE("as_float(10)", T_FLOAT(10.0));
    TEST_CASE("as_float(false)", T_FLOAT(0.0));
    TEST_CASE("as_float('-23.2356')", T_FLOAT(-23.2356));
    TEST_CASE("as_float('23.56')", T_FLOAT(23.56));
    TEST_CASE("as_bool(10)", T_BOOL(true));
    TEST_CASE("as_bool(0)", T_BOOL(false));
    TEST_CASE("as_bool(false)", T_BOOL(false));
    TEST_CASE("as_bool(true)", T_BOOL(true));
    TEST_CASE("as_string(10234)", T_STRING("10234"));
    TEST_CASE("as_string(true)", T_STRING("true"));
    TEST_CASE("as_string(false)", T_STRING("false"));
    TEST_CASE("as_string(-120)", T_STRING("-120"));
    TEST_CASE("as_string(120.234)", T_STRING("120.234000"));
    TEST_CASE("as_string(-120.234)", T_STRING("-120.234000"));

    TEST_CASE_LABEL("Unit type");
    TEST_CASE("?", T_UNIT());
    TEST_CASE("a = ?", T_UNIT());

    TEST_CASE_LABEL("String type");
    TEST_CASE("'Hello, World'", T_STRING("Hello, World"));
    TEST_CASE("a = 'Hello, World'", T_STRING("Hello, World"));
    TEST_CASE("println('Hello, World')", T_STRING("Hello, World"));
    TEST_CASE("print('Hello, World\\n')", T_STRING("Hello, World\n"));

    TEST_CASE_LABEL("Functions");
    TEST_CASE("f = \\(a, b) -> a + b; f(10, 20)", T_INT(30));
    TEST_CASE("f = \\() -> 100; f()", T_INT(100));
    TEST_CASE("f = \\(x) -> { x * 2 }; f(10)", T_INT(20));
    TEST_CASE("f = \\(x) -> { a = 10; x + a }; f(5)", T_INT(15));
    TEST_CASE("f = \\(a, cb) -> cb(a); f(10, \\(x) -> x * 2)", T_INT(20));
    TEST_CASE("make_adder = \\(x) -> \\(y) -> x + y; add_5 = make_adder(5); add_5(10)", T_INT(15));
    TEST_CASE("x = 10; f = \\(x) -> x * 2; f(20) + x", T_INT(50));
    TEST_CASE("x = 10; f = \\() -> x; x = 20; f()", T_INT(20));
    TEST_CASE("f = \\(x) -> x; if true { y = 100; f(y) }", T_INT(100));

    TEST_CASE_LABEL("Hashmaps");
    TEST_CASE(
        "m = map_init();"
        "len(m);",
        T_INT(0)
    );
    TEST_CASE(
        "m = map_init();"
        "map_set(m, 1, 'Hello, World');"
        "map_get(m, 1)",
        T_STRING("Hello, World")
    );
    TEST_CASE(
        "m = map_init();"
        "map_set(m, 1, 'Hello, World');"
        "map_get(m, 2)",
        T_UNIT()
    );
    TEST_CASE(
        "m = map_init();"
        "map_set(m, 1, 'Hello, World');",
        T_STRING("Hello, World")
    );
    TEST_CASE(
        "m = map_init();"
        "map_set(m, 'width', '3rem');"
        "map_set(m, 'height', '3rem');"
        "map_set(m, 'z-index', 999);"
        "map_del(m, 'height')",
        T_BOOL(true)
    );
    TEST_CASE(
        "m = map_init();"
        "map_set(m, 'width', '3rem');"
        "map_set(m, 'height', '3rem');"
        "map_set(m, 'z-index', 999);"
        "map_del(m, 'Height')",
        T_BOOL(false)
    );
    TEST_CASE(
        "m = map_init();"
        "map_set(m, 'width', '3rem');"
        "map_set(m, 'height', '3rem');"
        "map_set(m, 'z-index', 999);"
        "map_clear(m);"
        "len(m)",
        T_INT(0)
    );
    TEST_CASE("m = map_init();map_set(m, 'width', '3rem');map_set(m, 'height', '3rem');map_set(m, 'z-index', 999);map_del(m, 'Height')", T_BOOL(false));
    TEST_CASE("m = map_init();map_set(m, 'width', '3rem');map_set(m, 'height', '3rem');map_set(m, 'z-index', 999);map_clear(m);len(m)", T_INT(0));

    TEST_CASE_LABEL("Arrays");
    TEST_CASE("a = []; len(a)", T_INT(0));
    TEST_CASE("a = [1, 2, 'three']; len(a)", T_INT(3));
    TEST_CASE("a = [1, 2, 'three']; a[0]", T_INT(1));
    TEST_CASE("a = [1, 2, 'three']; a[2]", T_STRING("three"));
    TEST_CASE("a = [1]; append(a, 2); len(a)", T_INT(2));
    TEST_CASE("a = [1]; append(a, 2); a[1]", T_INT(2));

    TEST_CASE_LABEL("Return");
    TEST_CASE("f = \\() -> { return 42; 100 }; f()", T_INT(42));
    TEST_CASE("f = \\(x) -> { if x == 1 { return 10 } return 20 }; f(1)", T_INT(10));
    TEST_CASE("f = \\(x) -> { if x == 1 { return 10 } return 20 }; f(2)", T_INT(20));
    TEST_CASE("f = \\(x) -> { while x < 10 { if x == 5 { return x } x += 1 } return 0 }; f(0)", T_INT(5));
    TEST_CASE("f = \\(x) -> { while x < 10 { if x == 5 { return x } x += 1 } return 0 }; f(6)", T_INT(0));
    TEST_CASE("f = \\() -> { return; 100 }; f()", T_UNIT());

    TEST_CASE_LABEL("Strings");
    TEST_CASE("a = 'Hello World'; a[0]", T_STRING("H"));
    TEST_CASE("'Hello World'[6]", T_STRING("W"));

    if (errors >= 1) {
        fprintf(stderr, "\n\033[0;31mfailed\033[0m with \033[1;31m%ld\033[0m errors; \033[1;34m%ld/%ld\033[0m passed\n", errors, success, tests_count);
    } else {
        fprintf(stderr, "\nAll \033[1;36m%ld\033[0m tests \033[1;32mpass successfully\033[0m\n", success);
    }
}
