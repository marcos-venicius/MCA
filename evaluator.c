#include <math.h>
#include <assert.h>

#include "./evaluator.h"

static double calculate_factorial(double number) {
    if (number < 0 && number == (int)number) return NAN;
    
    return tgamma(number + 1.0);
}

static double evaluate_expression_impl(M_Expression *expression) {
    assert(expression != NULL && "evaluate_expression_impl: expression cannot be null");

    if (expression->kind == M_EK_NUMBER) return expression->number;

    if (expression->kind == M_EK_UNARY) {
        switch (expression->unary.op) {
            case M_UNARY_MINUS_OP: return -evaluate_expression_impl(expression->unary.operand);
            case M_UNARY_FACTORIAL_OP: return calculate_factorial(evaluate_expression_impl(expression->unary.operand));
            default:
                assert(0 && "evaluate_expression_impl: invalid unary expression operator");
        }
    }

    assert(expression->kind == M_EK_BINARY && "evaluate_expression_impl: should be a binary expression");

    switch (expression->binary.op) {
        case M_BINARY_PLUS_OP: return evaluate_expression_impl(expression->binary.left) + evaluate_expression_impl(expression->binary.right);
        case M_BINARY_TIMES_OP: return evaluate_expression_impl(expression->binary.left) * evaluate_expression_impl(expression->binary.right);
        case M_BINARY_DIVIDE_OP: return evaluate_expression_impl(expression->binary.left) / evaluate_expression_impl(expression->binary.right);
        case M_BINARY_SUBTRACT_OP: return evaluate_expression_impl(expression->binary.left) - evaluate_expression_impl(expression->binary.right);
        case M_BINARY_MOD_OP: return fmod(evaluate_expression_impl(expression->binary.left), evaluate_expression_impl(expression->binary.right));
        case M_BINARY_POW_OP: return pow(evaluate_expression_impl(expression->binary.left), evaluate_expression_impl(expression->binary.right));
        default:
            assert(0 && "evaluate_expression_impl: invalid binary expression operator");
    }
}

double evaluate_expression(M_Expression *expression) {
    if (expression == NULL) return 0;

    return evaluate_expression_impl(expression);
}
