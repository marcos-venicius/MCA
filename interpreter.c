#include <math.h>
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "./interpreter.h"
#include "./ast.h"
#include "./ht.h"

#define TRUE 1.0
#define FALSE 0.0


#define ctrl_normal(v) ((M_Eval_Result){ .value = v, .flow = M_CTRL_NORMAL })
#define ctrl_break(x) ((M_Eval_Result){ .value = x.value, .flow = M_CTRL_BREAK })
#define ctrl_break_value(x) ((M_Eval_Result){ .value = x, .flow = M_CTRL_BREAK })
#define ctrl_unwrap(c) (c.value)
#define ctrl_negate(x) ((M_Eval_Result){ .value = -x.value, .flow = x.flow })

//
// Here is the whole state of the interpreter.
// It's global, so, you cannot run, at this moment multiple programs.
// For now, let's focuse in run at least one single program well.
//
static M_Interpreter *interpreter = NULL;

typedef double (*M_Fn_C_Impl)(M_Expression *arguments[], int arguments_count);

// BUILTIN FUNCTION DECLARATIONS ----------------------------------------------------------------------------------------------------
static double __builtin_mca_pi(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_e(M_Expression *arguments[], int arguments_count);

static double __builtin_mca_abs(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_max(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_min(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_sin(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_cos(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_rad(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_deg(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_tan(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_sqrt(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_log(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_log10(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_exp(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_floor(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_ceil(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_round(M_Expression *arguments[], int arguments_count);

static double __builtin_mca_println(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_print(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_exit(M_Expression *arguments[], int arguments_count);

static double __builtin_mca_time(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_year(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_month(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_date(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_day(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_hour(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_minute(M_Expression *arguments[], int arguments_count);
static double __builtin_mca_second(M_Expression *arguments[], int arguments_count);
// BUILTIN FUNCTION DECLARATIONS ----------------------------------------------------------------------------------------------------

typedef struct {
    const char *name;
    int         name_length;
    int         arguments_count;
    M_Fn_C_Impl c_impl;
} M_Fn_Binding;

static M_Fn_Binding builtin_functions_bindings[] = {
    // Math related
    { "pi",    2, 0, &__builtin_mca_pi }, // TODO: should it become a constant variable?
    { "e",     1, 0, &__builtin_mca_e },  // TODO: should it become a constant variable?
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
    { "exp",   3, 1, &__builtin_mca_exp },
    { "floor", 5, 1, &__builtin_mca_floor },
    { "ceil",  4, 1, &__builtin_mca_ceil },
    { "round", 5, 1, &__builtin_mca_round },

    // I/O / System related
    { "println", 7, -1, &__builtin_mca_println },
    { "print",   5, -1, &__builtin_mca_print },
    { "exit",    4,  1, &__builtin_mca_exit },

    // datetime related
    { "time",   4, 0, &__builtin_mca_time },
    { "year",   4, 1, &__builtin_mca_year },
    { "month",  5, 1, &__builtin_mca_month },
    { "date",   4, 1, &__builtin_mca_date },
    { "day",    3, 1, &__builtin_mca_day },
    { "hour",   4, 1, &__builtin_mca_hour },
    { "minute", 6, 1, &__builtin_mca_minute },
    { "second", 6, 1, &__builtin_mca_second },
};

static int builtin_functions_bindings_length = sizeof(builtin_functions_bindings) / sizeof(M_Fn_Binding);

static M_Eval_Result calculate_factorial(M_Eval_Result r) {
    if (r.value < 0 && r.value == (int)r.value) return (M_Eval_Result){.flow = r.flow, .value = NAN};
    
    double x = tgamma(r.value + 1.0);

    return (M_Eval_Result){.flow = r.flow, .value = x};
}

static double evaluate_function_call_expression(M_Expression *expr) {
    for (int i = 0; i < builtin_functions_bindings_length; i++) {
        M_Fn_Binding signature = builtin_functions_bindings[i];

        if (signature.name_length != expr->call.fn_name_length) continue;

        if (strncmp(signature.name, expr->call.fn_name, signature.name_length) != 0) continue;

        if (signature.arguments_count == -1)
            return signature.c_impl(expr->call.arguments, expr->call.arguments_length);

        if (expr->call.arguments_length > signature.arguments_count) {
            // TODO: implement better error reporting at evaluation level
            fprintf(stderr, "\033[1;31merror:\033[0m to many arguments %s(...). expected %d but got %d\n", signature.name, signature.arguments_count, expr->call.arguments_length);
            exit(1);
        } else if (expr->call.arguments_length < signature.arguments_count) {
            // TODO: implement better error reporting at evaluation level
            fprintf(stderr, "\033[1;31merror:\033[0m to few arguments %s(...). expected %d but got %d\n", signature.name, signature.arguments_count, expr->call.arguments_length);
            exit(1);
        }

        return signature.c_impl(expr->call.arguments, expr->call.arguments_length);
    }

    // TODO: implement better error reporting at evaluation level
    fprintf(stderr, "\033[1;31merror:\033[0m function '%.*s' does not exists\n", expr->call.fn_name_length, expr->call.fn_name);
    exit(1);
}

static void enter_new_environment() {
    M_Interpreter_Environment *new_env = malloc(sizeof(M_Interpreter_Environment));
    new_env->variables = ht_init(sizeof(double)); // TODO: later this should be a struct M_Value that can hold different datatypes, not only numbers
    new_env->parent = interpreter->current_environment;

    interpreter->current_environment = new_env;
}

static void destroy_current_environment() {
    M_Interpreter_Environment *current_env = interpreter->current_environment;

    if (current_env == NULL) return;

    ht_free(current_env->variables);

    interpreter->current_environment = current_env->parent;

    free(current_env);
}

static void *get_variable_from_environment(M_Interpreter_Environment *env, const char *key) {
    if (env == NULL) return NULL;

    double *value = ht_find(env->variables, key);

    if (value != NULL) return value;

    return get_variable_from_environment(env->parent, key);
}

M_Eval_Result evaluate_expression(M_Expression *expression) {
    assert(expression != NULL && "evaluate_expression_impl: expression cannot be null");

    switch (expression->kind) {
        case M_EK_EXPRESSION_LIST: assert(0 && "evaluate_expression: case M_EK_EXPRESSION_LIST. should never happen. this is handled in an upper level");
        case M_EK_NUMBER: return ctrl_normal(expression->number);
        case M_EK_ID: {
            char *key = strndup(expression->id.value, expression->id.value_length);

            double *value = get_variable_from_environment(interpreter->current_environment, key);

            if (value == NULL) {
                // TODO: implement better error reporting at evaluation level
                fprintf(stderr, "\033[1;31merror:\033[0m variable '%s' does not exists\n", key);
                exit(1);
            }

            free(key);

            return ctrl_normal(*value);
        };
        case M_EK_ASSIGN: {
            M_Eval_Result result = evaluate_expression(expression->assign.right);
            // TODO: shouldn't I just keep the expression and lazy-evaluate it?
            double value = ctrl_unwrap(result);

            // @Leak TODO: we're not cleaning this
            const char *key = strndup(expression->assign.name.value, expression->assign.name.length);

            ht_add(interpreter->current_environment->variables, key, &value);

            return result;
        };
        case M_EK_UNARY: {
            switch (expression->unary.op) {
                case M_UNARY_MINUS_OP: return ctrl_negate(evaluate_expression(expression->unary.operand));
                case M_UNARY_FACTORIAL_OP: return calculate_factorial(evaluate_expression(expression->unary.operand));
            }

            assert(0 && "evaluate_expression_impl: invalid unary expression operator");
        } break;
        case M_EK_CALL: return ctrl_normal(evaluate_function_call_expression(expression));
        case M_EK_BINARY: {
            double left = ctrl_unwrap(evaluate_expression(expression->binary.left));
            double right = ctrl_unwrap(evaluate_expression(expression->binary.right));

            switch (expression->binary.op) {
                case M_BINARY_PLUS_OP: return ctrl_normal(left + right);
                case M_BINARY_TIMES_OP: return ctrl_normal(left * right);
                case M_BINARY_DIVIDE_OP: return ctrl_normal(left / right);
                case M_BINARY_SUBTRACT_OP: return ctrl_normal(left - right);
                case M_BINARY_MOD_OP: return ctrl_normal(fmod(left, right));
                case M_BINARY_POW_OP: return ctrl_normal(pow(left, right));

                case M_BINARY_EQUAL_OP: return ctrl_normal(left == right ? TRUE : FALSE);
                case M_BINARY_NOT_EQUAL_OP: return ctrl_normal(left != right ? TRUE : FALSE);
                case M_BINARY_GT_OP: return ctrl_normal(left > right ? TRUE : FALSE);
                case M_BINARY_LT_OP: return ctrl_normal(left < right ? TRUE : FALSE);
                case M_BINARY_GTE_OP: return ctrl_normal(left >= right ? TRUE : FALSE);
                case M_BINARY_LTE_OP: return ctrl_normal(left <= right ? TRUE : FALSE);
            }

            assert(0 && "evaluate_expression_impl: invalid binary expression operator");
        } break;
        case M_EK_IF: {
            M_Eval_Result condition_result = evaluate_expression(expression->if_expr.condition);
            M_Eval_Result last_evaluated_expression = {0};

            enter_new_environment();

            if (ctrl_unwrap(condition_result) != 0) {
                M_Expression_Block *current = expression->if_expr.then_block;

                M_Eval_Result last_evaluated_expression = {0};

                while (current != NULL) {
                    if (current->expr != NULL) {
                        last_evaluated_expression = evaluate_expression(current->expr);
                    }

                    // propagate break flow if exists
                    if (last_evaluated_expression.flow == M_CTRL_BREAK) break;

                    current = current->next;
                }
            } else {
                M_Expression_Block *current = expression->if_expr.else_block;

                while (current != NULL) {
                    if (current->expr != NULL) {
                        last_evaluated_expression = evaluate_expression(current->expr);
                    }

                    // propagate break flow if exists
                    if (last_evaluated_expression.flow == M_CTRL_BREAK) break;

                    current = current->next;
                }
            }

            destroy_current_environment();

            return last_evaluated_expression;
        } break;
        case M_EK_LOOP: {
            M_Eval_Result last_evaluated_expression = {0};

            enter_new_environment();

            while (1) {
                if (expression->loop.condition != NULL) {
                    if (ctrl_unwrap(evaluate_expression(expression->loop.condition)) == 0) break;
                }

                M_Expression_Block *current = expression->loop.block;

                while (current != NULL) {
                    if (current->expr != NULL) {
                        last_evaluated_expression = evaluate_expression(current->expr);
                    }

                    if (last_evaluated_expression.flow == M_CTRL_BREAK) {
                        last_evaluated_expression = ctrl_normal(last_evaluated_expression.value);
                        goto m_ek_loop_out;
                    }

                    current = current->next;
                }
            }
m_ek_loop_out:
            destroy_current_environment();

            return last_evaluated_expression;
        } break;
        case M_EK_BREAK:
            return expression->expr != NULL ? ctrl_break(evaluate_expression(expression->expr)) : ctrl_break_value(0);
    }

    return ctrl_normal(0);
}

M_Interpreter *m_interpreter_create(M_Ast *program) {
    interpreter = malloc(sizeof(M_Interpreter));

    interpreter->program = program;
    interpreter->global_environment = malloc(sizeof(M_Interpreter_Environment));
    // TODO: for now, we can only work with numbers
    interpreter->global_environment->variables = ht_init(sizeof(double)); // TODO: later this should be a struct M_Value that can hold different datatypes, not only numbers
    interpreter->global_environment->parent = NULL;
    interpreter->current_environment = interpreter->global_environment;

    return interpreter;
}

double m_interpreter_run(M_Interpreter *interpreter) {
    if (interpreter->program == NULL) return 0;

    double last_evaluated_expression = 0;

    for (int i = 0; i < interpreter->program->expressions_array_length; i++) {
        M_Expression *expr = interpreter->program->expressions_array[i];

        if (expr != NULL) {
            M_Eval_Result r = evaluate_expression(expr);

            if (r.flow == M_CTRL_BREAK) {
                // TODO: implement better error reporting at evaluation level
                fprintf(stderr, "\033[1;31merror:\033[0m cannot use 'break' outside of a loop\n");
                exit(1);
            }

            last_evaluated_expression = r.value;
        }
    }

    return last_evaluated_expression;
}

void m_interpreter_free(M_Interpreter *interpreter) {
    ht_free(interpreter->global_environment->variables);
    free(interpreter->global_environment);
    ast_free(interpreter->program);
    free(interpreter);

    interpreter = NULL;
}

// BUILTIN FUNCTION IMPLEMENTATIONS ----------------------------------------------------------------------------------------------------
static double __builtin_mca_pi(M_Expression *arguments[], int arguments_count) {
    (void)arguments;
    (void)arguments_count;

    return M_PI;
}

static double __builtin_mca_e(M_Expression *arguments[], int arguments_count) {
    (void)arguments;
    (void)arguments_count;

    return M_E;
}

static double __builtin_mca_abs(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return fabs(a0);
}

static double __builtin_mca_max(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));
    double a1 = ctrl_unwrap(evaluate_expression(arguments[1]));

    if (a0 > a1) return a0;

    return a1;
}

static double __builtin_mca_min(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));
    double a1 = ctrl_unwrap(evaluate_expression(arguments[1]));

    if (a0 < a1) return a0;

    return a1;
}

static double __builtin_mca_sin(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return sin(a0);
}

static double __builtin_mca_cos(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return cos(a0);
}

static double __builtin_mca_rad(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return a0 * (M_PI / 180.0);
}

static double __builtin_mca_deg(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return a0 * (180.0 / M_PI);
}

static double __builtin_mca_tan(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return tan(a0);
}

static double __builtin_mca_sqrt(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return sqrt(a0);
}

static double __builtin_mca_log(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return log(a0);
}

static double __builtin_mca_log10(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return log10(a0);
}

static double __builtin_mca_exp(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return exp(a0);
}

static double __builtin_mca_floor(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return floor(a0);
}

static double __builtin_mca_ceil(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return ceil(a0);
}

static double __builtin_mca_round(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = ctrl_unwrap(evaluate_expression(arguments[0]));

    return round(a0);
}

static double __builtin_mca_println(M_Expression *arguments[], int arguments_count) {
    double last_value = __builtin_mca_print(arguments, arguments_count);

    printf("\n");

    return last_value;
}

static double __builtin_mca_print(M_Expression *arguments[], int arguments_count) {
    double last_value = 0.0;

    for (int i = 0; i < arguments_count; i++) {
        if (i > 0) printf(" ");

        last_value = ctrl_unwrap(evaluate_expression(arguments[i]));

        printf("%f", last_value);
    }

    return last_value;
}

static double __builtin_mca_exit(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    exit((int)ctrl_unwrap(evaluate_expression(arguments[0])));
}

static double __builtin_mca_time(M_Expression *arguments[], int arguments_count) {
    (void)arguments;
    (void)arguments_count;

    return (double)time(NULL);
}

static double __builtin_mca_year(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)ctrl_unwrap(evaluate_expression(arguments[0]));

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (double)(time_info->tm_year + 1900);
}

static double __builtin_mca_month(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)ctrl_unwrap(evaluate_expression(arguments[0]));

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (double)(time_info->tm_mon + 1);
}

static double __builtin_mca_date(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)ctrl_unwrap(evaluate_expression(arguments[0]));

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (double)time_info->tm_mday;
}

static double __builtin_mca_day(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)ctrl_unwrap(evaluate_expression(arguments[0]));

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (double)time_info->tm_wday;
}

static double __builtin_mca_hour(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)ctrl_unwrap(evaluate_expression(arguments[0]));

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (double)time_info->tm_hour;
}

static double __builtin_mca_minute(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)ctrl_unwrap(evaluate_expression(arguments[0]));

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (double)time_info->tm_min;
}

static double __builtin_mca_second(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)ctrl_unwrap(evaluate_expression(arguments[0]));

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (double)time_info->tm_sec;
}