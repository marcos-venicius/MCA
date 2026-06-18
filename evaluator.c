#include <math.h>
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "./evaluator.h"

#define TRUE 1.0
#define FALSE 0.0

typedef double (*M_Fn_C_Impl)(M_Expression *arguments[]);

// BUILTIN FUNCTION DECLARATIONS ----------------------------------------------------------------------------------------------------
static double __builtin_mca_pi(M_Expression *arguments[]);
static double __builtin_mca_e(M_Expression *arguments[]);

static double __builtin_mca_abs(M_Expression *arguments[]);
static double __builtin_mca_max(M_Expression *arguments[]);
static double __builtin_mca_min(M_Expression *arguments[]);
static double __builtin_mca_sin(M_Expression *arguments[]);
static double __builtin_mca_cos(M_Expression *arguments[]);
static double __builtin_mca_rad(M_Expression *arguments[]);
static double __builtin_mca_deg(M_Expression *arguments[]);
static double __builtin_mca_tan(M_Expression *arguments[]);
static double __builtin_mca_sqrt(M_Expression *arguments[]);
static double __builtin_mca_log(M_Expression *arguments[]);
static double __builtin_mca_log10(M_Expression *arguments[]);
static double __builtin_mca_exp(M_Expression *arguments[]);
static double __builtin_mca_floor(M_Expression *arguments[]);
static double __builtin_mca_ceil(M_Expression *arguments[]);
static double __builtin_mca_round(M_Expression *arguments[]);

static double __builtin_mca_if(M_Expression *arguments[]);
// BUILTIN FUNCTION DECLARATIONS ----------------------------------------------------------------------------------------------------

typedef struct {
    const char *name;
    int         name_length;
    int         arguments_count;
    M_Fn_C_Impl c_impl;
} M_Fn_Binding;

static M_Fn_Binding builtin_functions_bindings[] = {
    // constants (TODO: should we have variables?)
    { "pi", 2, 0, &__builtin_mca_pi },
    { "e",  1, 0, &__builtin_mca_e },

    // functions
    { "abs",   3, 1, &__builtin_mca_abs },
    { "max",   3, 2, &__builtin_mca_max },
    { "min",   3, 2, &__builtin_mca_min },
    { "sin",   3, 1, &__builtin_mca_sin },
    { "cos",   3, 1, &__builtin_mca_cos },
    { "tan",   3, 1, &__builtin_mca_tan },
    { "rad",   3, 1, &__builtin_mca_rad },
    { "deg",   3, 1, &__builtin_mca_deg },
    { "sqrt",  4, 1, &__builtin_mca_sqrt },
    { "log",   3, 1, &__builtin_mca_log },
    { "log10", 5, 1, &__builtin_mca_log10 },
    { "exp",  3, 1, &__builtin_mca_exp },
    { "floor", 5, 1, &__builtin_mca_floor },
    { "ceil",  4, 1, &__builtin_mca_ceil },
    { "round", 5, 1, &__builtin_mca_round },

    // conditionals
    { "if", 2, 3, &__builtin_mca_if },
};

static int builtin_functions_bindings_length = sizeof(builtin_functions_bindings) / sizeof(M_Fn_Binding);

static double calculate_factorial(double number) {
    if (number < 0 && number == (int)number) return NAN;
    
    return tgamma(number + 1.0);
}

static double evaluate_function_call_expression(M_Expression *expr) {
    for (int i = 0; i < builtin_functions_bindings_length; i++) {
        M_Fn_Binding signature = builtin_functions_bindings[i];

        if (signature.name_length != expr->call.fn_name_length) continue;

        if (strncmp(signature.name, expr->call.fn_name, signature.name_length) != 0) continue;

        if (expr->call.arguments_length > signature.arguments_count) {
            // TODO: implement better error reporting at evaluation level
            fprintf(stderr, "\033[1;31merror:\033[0m to many arguments %s(...). expected %d but got %d\n", signature.name, signature.arguments_count, expr->call.arguments_length);
            exit(1);
        } else if (expr->call.arguments_length < signature.arguments_count) {
            // TODO: implement better error reporting at evaluation level
            fprintf(stderr, "\033[1;31merror:\033[0m to few arguments %s(...). expected %d but got %d\n", signature.name, signature.arguments_count, expr->call.arguments_length);
            exit(1);
        }

        return signature.c_impl(expr->call.arguments);
    }

    // TODO: implement better error reporting at evaluation level
    fprintf(stderr, "\033[1;31merror:\033[0m function '%.*s' does not exists\n", expr->call.fn_name_length, expr->call.fn_name);
    exit(1);
}

static double evaluate_expression_impl(M_Expression *expression) {
    assert(expression != NULL && "evaluate_expression_impl: expression cannot be null");

    if (expression->kind == M_EK_NUMBER) return expression->number;

    if (expression->kind == M_EK_UNARY) {
        switch (expression->unary.op) {
            case M_UNARY_MINUS_OP: return -evaluate_expression_impl(expression->unary.operand);
            case M_UNARY_FACTORIAL_OP: return calculate_factorial(evaluate_expression_impl(expression->unary.operand));
        }

        assert(0 && "evaluate_expression_impl: invalid unary expression operator");
    }

    if (expression->kind == M_EK_CALL) return evaluate_function_call_expression(expression);

    assert(expression->kind == M_EK_BINARY && "evaluate_expression_impl: should be a binary expression");

    double left = evaluate_expression_impl(expression->binary.left);
    double right = evaluate_expression_impl(expression->binary.right);

    switch (expression->binary.op) {
        case M_BINARY_PLUS_OP: return left + right;
        case M_BINARY_TIMES_OP: return left * right;
        case M_BINARY_DIVIDE_OP: return left / right;
        case M_BINARY_SUBTRACT_OP: return left - right;
        case M_BINARY_MOD_OP: return fmod(left, right);
        case M_BINARY_POW_OP: return pow(left, right);

        case M_BINARY_EQUAL_OP: return left == right ? TRUE : FALSE;
        case M_BINARY_NOT_EQUAL_OP: return left != right ? TRUE : FALSE;
        case M_BINARY_GT_OP: return left > right ? TRUE : FALSE;
        case M_BINARY_LT_OP: return left < right ? TRUE : FALSE;
        case M_BINARY_GTE_OP: return left >= right ? TRUE : FALSE;
        case M_BINARY_LTE_OP: return left <= right ? TRUE : FALSE;
    }

    assert(0 && "evaluate_expression_impl: invalid binary expression operator");
}

double evaluate_expression(M_Expression *expression) {
    if (expression == NULL) return 0;

    return evaluate_expression_impl(expression);
}

// BUILTIN FUNCTION IMPLEMENTATIONS ----------------------------------------------------------------------------------------------------
static double __builtin_mca_pi(M_Expression *arguments[]) {
    (void)arguments;

    return M_PI;
}

static double __builtin_mca_e(M_Expression *arguments[]) {
    (void)arguments;

    return M_E;
}

static double __builtin_mca_abs(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return fabs(a0);
}

static double __builtin_mca_max(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);
    double a1 = evaluate_expression_impl(arguments[1]);

    if (a0 > a1) return a0;

    return a1;
}

static double __builtin_mca_min(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);
    double a1 = evaluate_expression_impl(arguments[1]);

    if (a0 < a1) return a0;

    return a1;
}

static double __builtin_mca_sin(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return sin(a0);
}

static double __builtin_mca_cos(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return cos(a0);
}

static double __builtin_mca_rad(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return a0 * (M_PI / 180.0);
}

static double __builtin_mca_deg(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return a0 * (180.0 / M_PI);
}

static double __builtin_mca_tan(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return tan(a0);
}

static double __builtin_mca_sqrt(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return sqrt(a0);
}

static double __builtin_mca_log(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return log(a0);
}

static double __builtin_mca_log10(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return log10(a0);
}

static double __builtin_mca_exp(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return exp(a0);
}

static double __builtin_mca_floor(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return floor(a0);
}

static double __builtin_mca_ceil(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return ceil(a0);
}

static double __builtin_mca_round(M_Expression *arguments[]) {
    double a0 = evaluate_expression_impl(arguments[0]);

    return round(a0);
}

static double __builtin_mca_if(M_Expression *arguments[]) {
    M_Expression *condition = arguments[0];
    M_Expression *then      = arguments[1];
    M_Expression *elze      = arguments[2];

    if (evaluate_expression(condition) != 0.0) return evaluate_expression(then);

    return evaluate_expression(elze);
}
