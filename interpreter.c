#include <math.h>
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "./interpreter.h"
#include "./ast.h"
#include "./ht.h"

#define m_value_zero() ((M_Value){ .type = M_T_INT, .as.integer = 0 })

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
    double val = 0.0;
    bool is_negative_int = false;

    if (r.value.type == M_T_INT) {
        val = (double)r.value.as.integer;
        if (r.value.as.integer < 0) {
            is_negative_int = true;
        }
    } else if (r.value.type == M_T_FLOAT) {
        val = r.value.as.floating;
        if (val < 0.0 && val == (int)val) {
            is_negative_int = true;
        }
    }

    if (is_negative_int) {
        return (M_Eval_Result){
            .flow = r.flow,
            .value = (M_Value){
                .type = M_T_FLOAT,
                .as.floating = NAN,
            }
        };
    }
    
    double x = tgamma(val + 1.0);

    M_Value_Union out;

    switch (r.value.type) {
        case M_T_INT: out.integer = (int64_t)x; break;
        case M_T_FLOAT: out.floating = x;
    }

    return (M_Eval_Result){
        .flow = r.flow,
        .value = (M_Value){
            .type = r.value.type,
            .as = out
        }
    };
}

static const char *m_value_type_name(M_Value_Type type) {
    switch (type) {
        case M_T_INT: return "int";
        case M_T_FLOAT: return "float";
        case (M_T_INT | M_T_FLOAT): return "int | float";
        default: return "unknown";
    }
}

static inline M_Eval_Result m_result_expect_type(M_Eval_Result result, M_Value_Type expected_type_mask) {
    if ((result.value.type & expected_type_mask) == 0) {
        // TODO: implement better error reporting at evaluation level
        fprintf(
            stderr,
            "\033[1;31merror:\033[0m unexpected data type. expected a '%s' but got a '%s'\n",
            m_value_type_name(expected_type_mask),
            m_value_type_name(result.value.type)
        );
        exit(1);
    }

    return result;
}

static double evaluate_binary_operation_on_doubles(M_Binary_Expression_Operator op, double left, double right) {
    switch (op) {
        case M_BINARY_PLUS_OP: return left + right;
        case M_BINARY_TIMES_OP: return left * right;
        case M_BINARY_DIVIDE_OP: return left / right;
        case M_BINARY_SUBTRACT_OP: return left - right;
        case M_BINARY_MOD_OP: return fmod(left, right);
        case M_BINARY_POW_OP: return pow(left, right);

        case M_BINARY_EQUAL_OP: return left == right;
        case M_BINARY_NOT_EQUAL_OP: return left != right;
        case M_BINARY_GT_OP: return left > right;
        case M_BINARY_LT_OP: return left < right;
        case M_BINARY_GTE_OP: return left >= right;
        case M_BINARY_LTE_OP: return left <= right;
    }

    assert(0 && "evaluate_binary_operation_on_doubles: missing implementation");
}

static int evaluate_m_value_as_internal_boolean(M_Value value) {
    switch (value.type) {
        case M_T_INT:
            return value.as.integer != 0;
        case M_T_FLOAT:
            return value.as.floating != 0;
        default:
            return -1;
    }
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
        case M_EK_INT:
            return (M_Eval_Result){ .value = (M_Value){ .type = M_T_INT , .as.integer = expression->integer }, .flow = M_CTRL_NORMAL };
        case M_EK_FLOAT:
            return (M_Eval_Result){ .value = (M_Value){ .type = M_T_FLOAT, .as.floating = expression->floating }, .flow = M_CTRL_NORMAL };
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
                    M_Eval_Result result = evaluate_expression(expression->unary.operand);
                    M_Value_Union out;

                    switch (result.value.type) {
                        case M_T_INT:
                            out.integer = -result.value.as.integer;
                            break;
                        case M_T_FLOAT:
                            out.floating = -result.value.as.floating;
                            break;
                        default:
                            // TODO: implement better error reporting at evaluation level
                            fprintf(
                                stderr,
                                "\033[1;31merror:\033[0m unexpected data type. expected a int or float but got a '%s'\n",
                                m_value_type_name(result.value.type)
                            );
                            exit(1);
                    }

                    return (M_Eval_Result){
                        .flow = result.flow,
                        .value = (M_Value){
                            .type = result.value.type,
                            .as = out
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
            M_Eval_Result left = m_result_expect_type(evaluate_expression(expression->binary.left), M_T_INT | M_T_FLOAT);
            M_Eval_Result right = m_result_expect_type(evaluate_expression(expression->binary.right), M_T_INT | M_T_FLOAT);

            M_Value_Type l_type = left.value.type;
            M_Value_Type r_type = right.value.type;

            if (l_type == M_T_FLOAT || r_type == M_T_FLOAT) {
                double l_val = (l_type == M_T_FLOAT) ? left.value.as.floating : (double)left.value.as.integer;
                double r_val = (r_type == M_T_FLOAT) ? right.value.as.floating : (double)right.value.as.integer;

                return (M_Eval_Result){
                    .value = {
                        .type = M_T_FLOAT,
                        .as.floating = evaluate_binary_operation_on_doubles(expression->binary.op,l_val, r_val)
                    },
                    .flow = M_CTRL_NORMAL
                };
            }

            // From now on, both should be integers

            int64_t l = left.value.as.integer;
            int64_t r = right.value.as.integer;
            double result = 0; 

            switch (expression->binary.op) {
                case M_BINARY_PLUS_OP:       result = l + r; break;
                case M_BINARY_TIMES_OP:      result = l * r; break;
                case M_BINARY_DIVIDE_OP:     result = (double)l / (double)r; break;
                case M_BINARY_SUBTRACT_OP:   result = l - r; break;
                case M_BINARY_MOD_OP:        result = l % r; break;
                case M_BINARY_POW_OP:        result = pow(l, r); break;
                case M_BINARY_EQUAL_OP:      result = l == r; break;
                case M_BINARY_NOT_EQUAL_OP:  result = l != r; break;
                case M_BINARY_GT_OP:         result = l > r; break;
                case M_BINARY_LT_OP:         result = l < r; break;
                case M_BINARY_GTE_OP:        result = l >= r; break;
                case M_BINARY_LTE_OP:        result = l <= r; break;
            }

            // Return integer or float based on whether there's a decimal remainder
            if (fmod(result, 1.0) != 0.0) {
                return (M_Eval_Result){
                    .value = {
                        .type = M_T_FLOAT,
                        .as.floating = result
                    },
                    .flow = M_CTRL_NORMAL
                };
            }

            return (M_Eval_Result){
                .value = {
                    .type = M_T_INT,
                    .as.integer = (int64_t)result
                },
                .flow = M_CTRL_NORMAL
            };
        } break;
        case M_EK_IF: {
            M_Eval_Result condition = evaluate_expression(expression->if_expr.condition);
            M_Eval_Result last_evaluated_expression = {
                .flow = M_CTRL_NORMAL,
                .value = m_value_zero(),
            };

            enter_new_environment();

            int evaluated_condition = evaluate_m_value_as_internal_boolean(condition.value);

            if (evaluated_condition == -1) {
                // TODO: implement better error reporting at evaluation level
                fprintf(
                    stderr,
                    "\033[1;31merror:\033[0m failed to check truthiness of '%s' data type on that 'if'\n",
                    m_value_type_name(condition.value.type)
                );
                exit(1);
            }

            if (evaluated_condition) {
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
            M_Eval_Result last_evaluated_expression = {
                .flow = M_CTRL_NORMAL,
                .value = m_value_zero()
            };

            enter_new_environment();

            while (1) {
                if (expression->loop.condition != NULL) {
                    M_Eval_Result condition = evaluate_expression(expression->loop.condition);

                    int evaluated_condition = evaluate_m_value_as_internal_boolean(condition.value);

                    if (evaluated_condition == -1) {
                        // TODO: implement better error reporting at evaluation level
                        fprintf(
                            stderr,
                            "\033[1;31merror:\033[0m failed to check truthiness of '%s' data type on that 'loop'\n",
                            m_value_type_name(condition.value.type)
                        );
                        exit(1);
                    }

                    if (!evaluated_condition) break;
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

    return (M_Value){ .type = M_T_FLOAT, .as.floating = M_PI };
}

static M_Value __builtin_mca_e(M_Expression *arguments[], int arguments_count) {
    (void)arguments;
    (void)arguments_count;

    return (M_Value){ .type = M_T_FLOAT, .as.floating = M_E };
}

static M_Value __builtin_mca_abs(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    M_Eval_Result arg = m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT | M_T_FLOAT);

    if (arg.value.type == M_T_INT) {
        return (M_Value){
            .type = M_T_INT,
            .as.integer = llabs(arg.value.as.integer)
        };
    } else {
        return (M_Value){
            .type = M_T_FLOAT,
            .as.floating = fabs(arg.value.as.floating)
        };
    }
}

static M_Value __builtin_mca_max(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    M_Eval_Result a0 = evaluate_expression(arguments[0]);
    M_Eval_Result a1 = evaluate_expression(arguments[1]);

    double a;
    double b;

    switch (a0.value.type) {
        case M_T_INT:
            a = a0.value.as.integer;
            break;
        case M_T_FLOAT:
            a = a0.value.as.floating;
            break;
        default:
            // TODO: implement better error reporting at evaluation level
            fprintf(
                stderr,
                "\033[1;31merror:\033[0m function 'max' does not accept arguments of data type '%s'\n",
                m_value_type_name(a0.value.type)
            );
            exit(1);
    }

    switch (a1.value.type) {
        case M_T_INT:
            b = a1.value.as.integer;
            break;
        case M_T_FLOAT:
            b = a1.value.as.floating;
            break;
        default:
            // TODO: implement better error reporting at evaluation level
            fprintf(
                stderr,
                "\033[1;31merror:\033[0m function 'max' does not accept arguments of data type '%s'\n",
                m_value_type_name(a1.value.type)
            );
            exit(1);
    }

    double r = b;

    if (a > b) r = a;

    if (fmod(r, 1) != 0) {
        return (M_Value){ .type = M_T_FLOAT, .as.floating = r };
    }

    return (M_Value){ .type = M_T_INT, .as.integer = r };
}

static M_Value __builtin_mca_min(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    M_Eval_Result x = m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT | M_T_FLOAT);
    M_Eval_Result y = m_result_expect_type(evaluate_expression(arguments[1]), M_T_INT | M_T_FLOAT);

    if (x.value.type == M_T_INT && y.value.type == M_T_INT) {
        int64_t a0 = x.value.as.integer;
        int64_t a1 = y.value.as.integer;

        return (M_Value){ .type = M_T_INT, .as.integer = (a0 < a1) ? a0 : a1 };
    }

    double a0 = (x.value.type == M_T_FLOAT) ? x.value.as.floating : (double)x.value.as.integer;
    double a1 = (y.value.type == M_T_FLOAT) ? y.value.as.floating : (double)y.value.as.integer;
    
    return (M_Value){ .type = M_T_FLOAT, .as.floating = (a0 < a1) ? a0 : a1 };
}

#define DEFINE_MATH_BUILTIN(func_name, c_math_function) \
    static M_Value __builtin_mca_##func_name(M_Expression *arguments[], int arguments_count) { \
        (void)arguments_count; \
        \
        M_Eval_Result arg = m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT | M_T_FLOAT); \
        \
        double input_val = (arg.value.type == M_T_FLOAT) \
                         ? arg.value.as.floating \
                         : (double)arg.value.as.integer; \
        \
        double result = c_math_function(input_val); \
        \
        if (fmod(result, 1.0) != 0.0) { \
            return (M_Value){ .type = M_T_FLOAT, .as.floating = result }; \
        } else { \
            return (M_Value){ .type = M_T_INT, .as.integer = (int64_t)result }; \
        } \
    }

static inline double calc_rad(double degrees) {
    return degrees * (M_PI / 180.0);
}

static inline double calc_deg(double radians) {
    return radians * (180.0 / M_PI);
}

DEFINE_MATH_BUILTIN(sin, sin)
DEFINE_MATH_BUILTIN(cos, cos)
DEFINE_MATH_BUILTIN(tan, tan)
DEFINE_MATH_BUILTIN(sqrt, sqrt)
DEFINE_MATH_BUILTIN(log, log)
DEFINE_MATH_BUILTIN(log10, log10)
DEFINE_MATH_BUILTIN(exp, exp)
DEFINE_MATH_BUILTIN(floor, floor)
DEFINE_MATH_BUILTIN(ceil, ceil)
DEFINE_MATH_BUILTIN(round, round)
DEFINE_MATH_BUILTIN(rad, calc_rad)
DEFINE_MATH_BUILTIN(deg, calc_deg)

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

        switch (last_value.type) {
            case M_T_INT:
                fprintf(interpreter->io_out, "%ld", last_value.as.integer);
                break;
            case M_T_FLOAT:
                fprintf(interpreter->io_out, "%f", last_value.as.floating);
                break;
        }
    }

    return last_value;
}

static M_Value __builtin_mca_exit(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    exit((int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT).value.as.integer);
}

static M_Value __builtin_mca_time(M_Expression *arguments[], int arguments_count) {
    (void)arguments;
    (void)arguments_count;

    return (M_Value){
        .type = M_T_INT,
        .as.integer = time(NULL)
    };
}

static M_Value __builtin_mca_year(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int64_t offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_INT,
        .as.integer = time_info->tm_year + 1900
    };
}

static M_Value __builtin_mca_month(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_INT,
        .as.integer = time_info->tm_mon + 1
    };
}

static M_Value __builtin_mca_date(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_INT,
        .as.integer = time_info->tm_mday
    };
}

static M_Value __builtin_mca_day(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_INT,
        .as.integer = time_info->tm_wday
    };
}

static M_Value __builtin_mca_hour(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_INT,
        .as.integer = time_info->tm_hour
    };
}

static M_Value __builtin_mca_minute(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_INT,
        .as.integer = time_info->tm_min
    };
}

static M_Value __builtin_mca_second(M_Expression *arguments[], int arguments_count) {
    (void)arguments_count;

    int offset = (int)m_result_expect_type(evaluate_expression(arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return (M_Value){
        .type = M_T_INT,
        .as.integer = time_info->tm_sec
    };
}
