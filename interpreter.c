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

#define m_value_zero() ((M_Value){ .type = M_T_NUMBER, .as.number = 0 })

//
// Here is the whole state of the interpreter.
// It's global, so, you cannot run, at this moment multiple programs.
// For now, let's focuse in run at least one single program well.
//
static M_Interpreter *interpreter = NULL;

typedef M_Value (*M_Fn_C_Impl)(M_Expression *arguments[], int arguments_count);

// BUILTIN FUNCTION DECLARATIONS ----------------------------------------------------------------------------------------------------
static M_Value __builtin_mca_pi(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_e(M_Expression *arguments[], int arguments_count);

static M_Value __builtin_mca_abs(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_max(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_min(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_sin(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_cos(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_rad(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_deg(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_tan(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_sqrt(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_log(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_log10(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_exp(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_floor(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_ceil(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_round(M_Expression *arguments[], int arguments_count);

static M_Value __builtin_mca_println(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_print(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_exit(M_Expression *arguments[], int arguments_count);

static M_Value __builtin_mca_time(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_year(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_month(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_date(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_day(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_hour(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_minute(M_Expression *arguments[], int arguments_count);
static M_Value __builtin_mca_second(M_Expression *arguments[], int arguments_count);
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
    if (r.value.as.number < 0 && r.value.as.number == (int)r.value.as.number) {
        return (M_Eval_Result){
            .flow = r.flow,
            .value = (M_Value){
                .type = M_T_NUMBER,
                .as.number = NAN,
            }
        };
    }
    
    double x = tgamma(r.value.as.number + 1.0);

    return (M_Eval_Result){
        .flow = r.flow,
        .value = (M_Value){
            .type = M_T_NUMBER,
            .as.number = x
        }
    };
}

static const char *m_value_type_name(M_Value_Type type) {
    switch (type) {
        case M_T_NUMBER: return "number";
    }

    return NULL;
}

static inline M_Eval_Result m_result_expect_type(M_Eval_Result result, M_Value_Type type) {
    if (result.value.type != type) {
        // TODO: implement better error reporting at evaluation level
        fprintf(
            stderr,
            "\033[1;31merror:\033[0m unexpected data type. expected a '%s' but got a '%s'\n",
            m_value_type_name(type),
            m_value_type_name(result.value.type)
        );
        exit(1);
    }

    return result;
}

static M_Value evaluate_function_call_expression(M_Expression *expr) {
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
    new_env->variables = ht_init(sizeof(M_Value));
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

static M_Value *get_variable_from_environment(M_Interpreter_Environment *env, const char *key) {
    if (env == NULL) return NULL;

    M_Value *value = ht_find(env->variables, key);

    if (value != NULL) return value;

    return get_variable_from_environment(env->parent, key);
}

static void set_variable_on_environment(M_Interpreter_Environment *env, const char *key, M_Value data) {
    // the variable doesn't exists on upper scopes, so we create one in the current scope
    if (env == NULL) {
        ht_add(interpreter->current_environment->variables, key, &data);

        return;
    }

    M_Value *value = ht_find(env->variables, key);

    // we find the variable at the current scope or on upper ones, so we update it
    if (value != NULL) {
        ht_add(env->variables, key, &data);
    }

    // we did not find the variable in this scope so we climb up
    set_variable_on_environment(env->parent, key, data);
}

M_Eval_Result evaluate_expression(M_Expression *expression) {
    assert(expression != NULL && "evaluate_expression_impl: expression cannot be null");

    switch (expression->kind) {
        case M_EK_EXPRESSION_LIST: assert(0 && "evaluate_expression: case M_EK_EXPRESSION_LIST. should never happen. this is handled in an upper level");
        case M_EK_NUMBER:
            return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = expression->number }, .flow = M_CTRL_NORMAL };
        case M_EK_ID: {
            char *key = strndup(expression->id.value, expression->id.value_length);

            M_Value *value = get_variable_from_environment(interpreter->current_environment, key);

            if (value == NULL) {
                // TODO: implement better error reporting at evaluation level
                fprintf(stderr, "\033[1;31merror:\033[0m variable '%s' does not exists\n", key);
                exit(1);
            }

            free(key);

            return (M_Eval_Result){ .value = *value, .flow = M_CTRL_NORMAL };
        };
        case M_EK_ASSIGN: {
            M_Eval_Result result = evaluate_expression(expression->assign.right);

            char *key = strndup(expression->assign.name.value, expression->assign.name.length);

            // will try to find the variable and update
            // if not find, will create one in the current scope
            set_variable_on_environment(interpreter->current_environment, key, result.value);

            free(key);

            return result;
        };
        case M_EK_UNARY: {
            switch (expression->unary.op) {
                case M_UNARY_MINUS_OP: {
                    M_Eval_Result result = m_result_expect_type(evaluate_expression(expression->unary.operand), M_T_NUMBER);

                    return (M_Eval_Result){
                        .flow = result.flow,
                        .value = (M_Value){
                            .type = M_T_NUMBER,
                            .as.number = -result.value.as.number
                        }
                    };
                } 
                case M_UNARY_FACTORIAL_OP: return calculate_factorial(evaluate_expression(expression->unary.operand));
            }

            assert(0 && "evaluate_expression_impl: invalid unary expression operator");
        } break;
        case M_EK_CALL:
            return (M_Eval_Result){ .value = evaluate_function_call_expression(expression), .flow = M_CTRL_NORMAL };
        case M_EK_BINARY: {
            // TODO: handle other cases like (string, string), etc
            double left = m_result_expect_type(evaluate_expression(expression->binary.left), M_T_NUMBER).value.as.number;
            double right = m_result_expect_type(evaluate_expression(expression->binary.right), M_T_NUMBER).value.as.number;

            switch (expression->binary.op) {
                case M_BINARY_PLUS_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left + right }, .flow = M_CTRL_NORMAL };
                case M_BINARY_TIMES_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left * right }, .flow = M_CTRL_NORMAL };
                case M_BINARY_DIVIDE_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left / right }, .flow = M_CTRL_NORMAL };
                case M_BINARY_SUBTRACT_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left - right }, .flow = M_CTRL_NORMAL };
                case M_BINARY_MOD_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = fmod(left, right) }, .flow = M_CTRL_NORMAL };
                case M_BINARY_POW_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = pow(left, right) }, .flow = M_CTRL_NORMAL };

                case M_BINARY_EQUAL_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left == right ? 1 : 0 }, .flow = M_CTRL_NORMAL };
                case M_BINARY_NOT_EQUAL_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left != right ? 1 : 0 }, .flow = M_CTRL_NORMAL };
                case M_BINARY_GT_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left > right ? 1 : 0 }, .flow = M_CTRL_NORMAL };
                case M_BINARY_LT_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left < right ? 1 : 0 }, .flow = M_CTRL_NORMAL };
                case M_BINARY_GTE_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left >= right ? 1 : 0 }, .flow = M_CTRL_NORMAL };
                case M_BINARY_LTE_OP: return (M_Eval_Result){ .value = (M_Value){ .type = M_T_NUMBER, .as.number = left <= right ? 1 : 0 }, .flow = M_CTRL_NORMAL };
            }

            assert(0 && "evaluate_expression_impl: invalid binary expression operator");
        } break;
        case M_EK_IF: {
            M_Eval_Result condition_result = evaluate_expression(expression->if_expr.condition);
            M_Eval_Result last_evaluated_expression = {0};

            enter_new_environment();

            if (m_result_expect_type(condition_result, M_T_NUMBER).value.as.number != 0) {
                M_Expression_Block *current = expression->if_expr.then_block;

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
                    if (m_result_expect_type(evaluate_expression(expression->loop.condition), M_T_NUMBER).value.as.number == 0) break;
                }

                M_Expression_Block *current = expression->loop.block;

                while (current != NULL) {
                    if (current->expr != NULL) {
                        last_evaluated_expression = evaluate_expression(current->expr);
                    }

                    if (last_evaluated_expression.flow == M_CTRL_BREAK) {
                        last_evaluated_expression = (M_Eval_Result){ .value = last_evaluated_expression.value, .flow = M_CTRL_NORMAL };

                        goto m_ek_loop_out;
                    }

                    current = current->next;
                }
            }
m_ek_loop_out:
            destroy_current_environment();

            return last_evaluated_expression;
        } break;
        case M_EK_BREAK: {
            if (expression->expr != NULL) {

                M_Eval_Result result = evaluate_expression(expression->expr);

                return (M_Eval_Result) { .value = result.value, .flow = M_CTRL_BREAK };
            }

            return (M_Eval_Result){ .value = m_value_zero(), .flow = M_CTRL_BREAK };
        };
    }

    return (M_Eval_Result){ .value = m_value_zero(), .flow = M_CTRL_NORMAL };
}

M_Interpreter *m_interpreter_create(M_Ast *program) {
    interpreter = malloc(sizeof(M_Interpreter));

    interpreter->program = program;
    interpreter->global_environment = malloc(sizeof(M_Interpreter_Environment));
    interpreter->io_in = stdin;
    interpreter->io_out = stdout;
    interpreter->io_err = stderr;
    interpreter->global_environment->variables = ht_init(sizeof(M_Value));
    interpreter->global_environment->parent = NULL;
    interpreter->current_environment = interpreter->global_environment;

    return interpreter;
}

void m_interpreter_set_stdin(M_Interpreter *interpreter, FILE *stream) {
    interpreter->io_in = stream;
}

void m_interpreter_set_stdout(M_Interpreter *interpreter, FILE *stream) {
    interpreter->io_out = stream;
}

void m_interpreter_set_stderr(M_Interpreter *interpreter, FILE *stream) {
    interpreter->io_err = stream;
}

M_Value m_interpreter_run(M_Interpreter *interpreter) {
    if (interpreter->program == NULL) return m_value_zero();

    M_Value last_evaluated_expression = m_value_zero();

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
static M_Value __builtin_mca_pi(M_Expression *arguments[], int arguments_count) {
    (void)arguments;
    (void)arguments_count;

    return (M_Value){ .type = M_T_NUMBER, .as.number = M_PI };
}

static M_Value __builtin_mca_e(M_Expression *arguments[], int arguments_count) {
    (void)arguments;
    (void)arguments_count;

    return (M_Value){ .type = M_T_NUMBER, .as.number = M_E };
}

static M_Value __builtin_mca_abs(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){ .type = M_T_NUMBER, .as.number = fabs(a0) };
}

static M_Value __builtin_mca_max(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;
    double a1 = m_result_expect_type(evaluate_expression(arguments[1]), M_T_NUMBER).value.as.number;

    double r = a1;

    if (a0 > a1) r = a0;

    return (M_Value){ .type = M_T_NUMBER, .as.number = r };
}

static M_Value __builtin_mca_min(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;
    double a1 = m_result_expect_type(evaluate_expression(arguments[1]), M_T_NUMBER).value.as.number;

    double r = a1;

    if (a0 < a1) r = a0;
    return (M_Value){ .type = M_T_NUMBER, .as.number = r };
}

static M_Value __builtin_mca_sin(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = sin(a0)
    };
}

static M_Value __builtin_mca_cos(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = cos(a0)
    };
}

static M_Value __builtin_mca_rad(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = a0 * (M_PI / 180.0)
    };
}

static M_Value __builtin_mca_deg(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = a0 * (180.0 / M_PI)
    };
}

static M_Value __builtin_mca_tan(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = tan(a0)
    };
}

static M_Value __builtin_mca_sqrt(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = sqrt(a0)
    };
}

static M_Value __builtin_mca_log(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = log(a0)
    };
}

static M_Value __builtin_mca_log10(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = log10(a0)
    };
}

static M_Value __builtin_mca_exp(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = exp(a0)
    };
}

static M_Value __builtin_mca_floor(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = floor(a0)
    };
}

static M_Value __builtin_mca_ceil(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = ceil(a0)
    };
}

static M_Value __builtin_mca_round(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;
    double a0 = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = round(a0)
    };
}

static M_Value __builtin_mca_println(M_Expression *arguments[], int arguments_count) {
    M_Value last_value = __builtin_mca_print(arguments, arguments_count);

    fprintf(interpreter->io_out, "\n");

    return last_value;
}

static M_Value __builtin_mca_print(M_Expression *arguments[], int arguments_count) {
    M_Value last_value = m_value_zero();

    for (int i = 0; i < arguments_count; i++) {
        if (i > 0) fprintf(interpreter->io_out, " ");

        last_value = evaluate_expression(arguments[i]).value;

        // TODO: handle other cases like string, etc
        fprintf(interpreter->io_out, "%f", last_value.as.number);
    }

    return last_value;
}

static M_Value __builtin_mca_exit(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    exit((int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number);
}

static M_Value __builtin_mca_time(M_Expression *arguments[], int arguments_count) {
    (void)arguments;
    (void)arguments_count;

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = (double)time(NULL)
    };
}

static M_Value __builtin_mca_year(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = (double)(time_info->tm_year + 1900)
    };
}

static M_Value __builtin_mca_month(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = (double)(time_info->tm_mon + 1)
    };
}

static M_Value __builtin_mca_date(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = (double)time_info->tm_mday
    };
}

static M_Value __builtin_mca_day(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = (double)time_info->tm_wday
    };
}

static M_Value __builtin_mca_hour(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = (double)time_info->tm_hour
    };
}

static M_Value __builtin_mca_minute(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = (double)time_info->tm_min
    };
}

static M_Value __builtin_mca_second(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_NUMBER).value.as.number;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_NUMBER,
        .as.number = (double)time_info->tm_sec
    };
}
