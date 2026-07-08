#include <math.h>
#include <assert.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <errno.h>
#include <inttypes.h>
#include <limits.h>
#include <libgen.h>

#include "./interpreter.h"
#include "./ast.h"
#include "./ht.h"
#include "./builtins/io.h"
#include "./builtins/map.h"
#include "./env.h"
#include "./colors.h"
#include "./constraints.h"
#include "./lexer.h"

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

#ifndef M_E
#define M_E 2.7182818284590452354
#endif


// [[typedefs]]
typedef M_Value (*M_Fn_C_Impl)(M_Interpreter *interpreter, M_Expression *caller, M_Expression *arguments[], int arguments_count);

// [[forward declarations]]
static M_Fn_C_Impl resolve_builtin_function(M_Expression *expr);
static M_Eval_Result evaluate_expression(M_Interpreter *interpreter, M_Expression *expression);
static M_Value __builtin_mca_map_parse_m_map_node_entry_helper(M_Map_Node_Entry *entry);

// [[macros]]
#define m_value_unit() ((M_Value){ .type = M_T_UNIT })
#define m_value_int(v) ((M_Value){ .type = M_T_INT, .as.integer = (v) })
#define m_value_float(v) ((M_Value){ .type = M_T_FLOAT, .as.floating = (v) })
#define m_value_string(v) ((M_Value){ .type = M_T_STRING, .as.string = (v) })
#define m_value_sized_string(v, s) ((M_Value){ .allocated = false, .type = M_T_STRING, .as.string.value = v, .as.string.value_length = s })
#define m_value_bool(v) ((M_Value){ .type = M_T_BOOL, .as.boolean = (v) })
#define m_value_true() ((M_Value){ .type = M_T_BOOL, .as.boolean = true })
#define m_value_false() ((M_Value){ .type = M_T_BOOL, .as.boolean = false })
#define m_value_map_it(it) ((M_Value){ .type = M_T_MAP_IT, .as.map_it = it })
#define m_value_fn(f) ((M_Value){ .type = M_T_FN, .as.fn = f })
#define m_string(s) ((M_String){ .value = s, .value_length = strlen(s) })

#define m_result_normal(v) (M_Eval_Result){ .flow = M_CTRL_NORMAL, .value = (v) }
#define m_result_break(v) (M_Eval_Result){ .flow = M_CTRL_BREAK, .value = (v) }
#define m_result_return(v) (M_Eval_Result){ .flow = M_CTRL_RETURN, .value = (v) }

#define PUBLIC
#define BUILTIN(name)  static M_Value name(M_Interpreter *interpreter, M_Expression *caller, M_Expression *arguments[], int arguments_count)

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

    M_Value_Union out = {0};

    switch (r.value.type) {
        case M_T_INT: out.integer = (int64_t)x; break;
        case M_T_FLOAT: out.floating = x;
        default: break; // unreachable
    }

    return (M_Eval_Result){
        .flow = r.flow,
        .value = (M_Value){
            .type = r.value.type,
            .as = out
        }
    };
}

#define HANDLE_M_VALUE_TYPE_NAME(T, name) if (t & T) { size += snprintf(buffer + size, 256 - size, "%s"name, (size > 0) ? " | " : ""); t &= ~T; }
static const char *m_value_type_name(M_Value_Type type) {
    static_assert(M_T_COUNT == 257, "m_value_type_name: missing M_Value_Type handling");

    int t = type;

    // @Leak TODO: this is never gonna be freed but
    // when this piece of code is called, normally the program is gonna exit
    // so, do we really care?
    char *buffer = malloc(256);
    if (!buffer) return NULL; 
    
    buffer[0] = '\0';
    int size = 0;

    HANDLE_M_VALUE_TYPE_NAME(M_T_INT, "int")
    HANDLE_M_VALUE_TYPE_NAME(M_T_FLOAT, "float")
    HANDLE_M_VALUE_TYPE_NAME(M_T_BOOL, "bool")
    HANDLE_M_VALUE_TYPE_NAME(M_T_UNIT, "unit")
    HANDLE_M_VALUE_TYPE_NAME(M_T_STRING, "string")
    HANDLE_M_VALUE_TYPE_NAME(M_T_MAP, "map")
    HANDLE_M_VALUE_TYPE_NAME(M_T_MAP_IT, "iter<map>")
    HANDLE_M_VALUE_TYPE_NAME(M_T_FN, "fn")
    HANDLE_M_VALUE_TYPE_NAME(M_T_ARRAY, "array")

    if (t != 0)
        snprintf(buffer + size, 256 - size, "%sunknown(%d)", (size > 0) ? " | " : "", t);

    return buffer;
}

static void m_interpreter_error(M_Expression *expr, const char *fmt, ...) {
    va_list args;
    va_start(args, fmt);

    if (expr->location.filename) {
        fprintf(stderr, "%s:%d:%d: \033[1;31mruntime error\033[0m: ", expr->location.filename, expr->location.line, expr->location.col);
    } else {
        fprintf(stderr, "%d:%d: \033[1;31mruntime error\033[0m: ", expr->location.line, expr->location.col);
    }

    vfprintf(stderr, fmt, args);
    fprintf(stderr, "\n");

    va_end(args);

    exit(1);
}

// static void m_interpreter_warn(M_Expression *expr, const char *fmt, ...) {
//     va_list args;
//     va_start(args, fmt);

//     if (expr->location.filename) {
//         fprintf(stderr, "%s:%d:%d: \033[1;33mruntime warn\033[0m: ", expr->location.filename, expr->location.line, expr->location.col);
//     } else {
//         fprintf(stderr, "%d:%d: \033[1;33mruntime warn\033[0m: ", expr->location.line, expr->location.col);
//     }

//     vfprintf(stderr, fmt, args);
//     fprintf(stderr, "\n");

//     va_end(args);
// }

static inline M_Eval_Result m_result_expect_type(M_Expression *expr, M_Eval_Result result, M_Value_Type expected_type_mask) {
    if ((result.value.type & expected_type_mask) == 0)
        m_interpreter_error(
            expr,
            "unexpected data type. expected a '%s' but got a '%s'",
            m_value_type_name(expected_type_mask),
            m_value_type_name(result.value.type)
        );

    return result;
}

static inline M_Value m_value_expect_type(M_Expression *expr, M_Value value, M_Value_Type expected_type_mask) {
    if ((value.type & expected_type_mask) == 0)
        m_interpreter_error(
            expr,
            "unexpected data type. expected a '%s' but got a '%s'",
            m_value_type_name(expected_type_mask),
            m_value_type_name(value.type)
        );

    return value;
}

static double evaluate_binary_operation_on_doubles(M_Binary_Expression_Operator op, double left, double right) {
    switch (op) {
        case M_BINARY_PLUS_OP:      return left + right;
        case M_BINARY_TIMES_OP:     return left * right;
        case M_BINARY_DIVIDE_OP:    return left / right;
        case M_BINARY_SUBTRACT_OP:  return left - right;
        case M_BINARY_MOD_OP:       return fmod(left, right);
        case M_BINARY_POW_OP:       return pow(left, right);

        case M_BINARY_EQUAL_OP:     return left == right;
        case M_BINARY_NOT_EQUAL_OP: return left != right;
        case M_BINARY_GT_OP:        return left > right;
        case M_BINARY_LT_OP:        return left < right;
        case M_BINARY_GTE_OP:       return left >= right;
        case M_BINARY_LTE_OP:       return left <= right;

        // these operations are handled inside `evaluate_binary_expression`
        case M_BINARY_AND_OP:       assert(0 && "evaluate_binary_operation_on_doubles: should never happen"); break;
        case M_BINARY_OR_OP:        assert(0 && "evaluate_binary_operation_on_doubles: should never happen"); break;
        case M_BINARY_OP_COUNT:     break;
    }

    static_assert(M_BINARY_OP_COUNT == 14, "evaluate_binary_operation_on_doubles: unhandled binary operator");
    return 0; // unreacheable
}

static int evaluate_m_value_as_internal_boolean(M_Value value) {
    static_assert(M_T_COUNT == 257, "evaluate_m_value_as_internal_boolean: missing M_Value_Type handling");

    switch (value.type) {
        case M_T_INT:   return value.as.integer != 0;
        case M_T_FLOAT: return value.as.floating != 0;
        case M_T_BOOL:  return value.as.boolean;

        default: return -1; // unreacheable
    }
}

static inline void acquire_environment(M_Environment *env) {
    if (env != NULL) env->ref_count++;
}

static void release_environment(M_Environment *env) {
    if (env == NULL) return;

    env->ref_count--;

    if (env->ref_count <= 0) {
        ht_free(env->variables);
        release_environment(env->parent);
        free(env);
    }
}

static M_Environment *create_new_environment(M_Environment *parent) {
    M_Environment *new_env = malloc(sizeof(M_Environment));
    new_env->parent = parent;
    new_env->variables = ht_init(sizeof(M_Value));
    new_env->ref_count = 1;

    acquire_environment(parent);

    return new_env;
}

static void enter_new_environment(M_Interpreter *interpreter) {
    M_Environment *new_env = malloc(sizeof(M_Environment));
    new_env->variables = ht_init(sizeof(M_Value));
    new_env->parent = interpreter->current_environment;
    new_env->ref_count = 1;

    interpreter->current_environment = new_env;
}

static void destroy_current_environment(M_Interpreter *interpreter) {
    M_Environment *current_env = interpreter->current_environment;

    if (current_env == NULL) return;

    ht_free(current_env->variables);

    interpreter->current_environment = current_env->parent;

    free(current_env);
}

static M_Value *get_variable_from_environment(M_Environment *env, const char *key) {
    if (env == NULL) return NULL;

    M_Value *value = ht_find(env->variables, key);

    if (value != NULL) return value;

    return get_variable_from_environment(env->parent, key);
}

static void define_variable_in_environment(M_Environment *env, const char *key, M_Value data) {
    if (env == NULL) return;

    ht_add(env->variables, key, &data);
}

static void set_variable_on_environment(M_Interpreter *interpreter, M_Environment *env, const char *key, M_Value data) {
    // the variable doesn't exists on upper scopes, so we create one in the current scope
    if (env == NULL) {
        ht_add(interpreter->current_environment->variables, key, &data);

        return;
    }

    M_Value *value = ht_find(env->variables, key);

    // we find the variable at the current scope or on upper ones, so we update it
    if (value != NULL) {
        ht_add(env->variables, key, &data);

        return;
    }

    // we did not find the variable in this scope so we climb up
    set_variable_on_environment(interpreter, env->parent, key, data);
}

static M_Eval_Result evaluate_block_expression(M_Interpreter *interpreter, M_Expression_Block *block) {
    M_Eval_Result last_result = m_result_normal(m_value_unit());

    M_Expression_Block *current = block;

    while (current != NULL) {
        if (current->expr != NULL) {
            last_result = evaluate_expression(interpreter, current->expr);

            if (last_result.flow != M_CTRL_NORMAL) return last_result;
        }

        current = current->next;
    }

    return last_result;
}

static M_Value evaluate_function_execution(M_Interpreter *interpreter, M_Expression *fn, M_Expression *call) {
    if (call->Call.arguments_length > fn->Fn.arguments_length) {
        m_interpreter_error(call, "too many arguments %s(...). expected %d but got %d", call->Call.fn_name.value, fn->Fn.arguments_length, call->Call.arguments_length);
    } else if (call->Call.arguments_length < fn->Fn.arguments_length) {
        m_interpreter_error(call, "too few arguments %s(...). expected %d but got %d", call->Call.fn_name.value, fn->Fn.arguments_length, call->Call.arguments_length);
    }

    // creating a new environment pointing the parent to the captured
    // event in the scope the function was defined
    M_Environment *fn_env     = create_new_environment(fn->Fn.closure_env);
    M_Environment *caller_env = interpreter->current_environment;

    // fill function parameters with given values
    for (int i = 0; i < fn->Fn.arguments_length; i++) {
        M_Eval_Result evaluated_argument = evaluate_expression(interpreter, call->Call.arguments[i]);

        define_variable_in_environment(fn_env, fn->Fn.arguments[i]->Id.value, evaluated_argument.value);
    }

    // evaluate the arguments to the function using the current scope
    // and only switch the scope to the closure scope when executing the function
    // code itself.
    interpreter->current_environment = fn_env;

    M_Eval_Result return_value = evaluate_block_expression(interpreter, fn->Fn.block);

    // restoring the environment
    interpreter->current_environment = caller_env;

    release_environment(fn_env);

    return return_value.value;
}

static M_Value evaluate_function_call_expression(M_Interpreter *interpreter, M_Expression *expr) {
    M_Fn_C_Impl fn = resolve_builtin_function(expr);

    if (fn != NULL)
        return fn(interpreter, expr, expr->Call.arguments, expr->Call.arguments_length);

    M_Value *var = get_variable_from_environment(interpreter->current_environment, expr->Call.fn_name.value);

    if (var == NULL)
        m_interpreter_error(expr, "function '%.*s' does not exists", expr->Call.fn_name.value_length, expr->Call.fn_name.value);

    if (var->type != M_T_FN)
        m_interpreter_error(expr, "you are trying to call '%s' that is a '%s', which it's not a function", expr->Call.fn_name, m_value_type_name(var->type));

    if (expr->Call.arguments_length > var->as.fn->Fn.arguments_length) {
        m_interpreter_error(expr, "too many arguments %s(...). expected %d but got %d", expr->Call.fn_name.value, var->as.fn->Fn.arguments_length, expr->Call.arguments_length);
    } else if (expr->Call.arguments_length < var->as.fn->Fn.arguments_length) {
        m_interpreter_error(expr, "too few arguments %s(...). expected %d but got %d", expr->Call.fn_name.value, var->as.fn->Fn.arguments_length, expr->Call.arguments_length);
    }

    return evaluate_function_execution(interpreter, var->as.fn, expr);
}

static M_Eval_Result evaluate_binary_expression(M_Interpreter *interpreter, M_Expression *expression) {
    switch (expression->Binary.op) {
        case M_BINARY_AND_OP:
            {
                M_Eval_Result left = m_result_expect_type(expression->Binary.left, evaluate_expression(interpreter, expression->Binary.left), M_T_BOOL | M_T_INT | M_T_FLOAT);

                switch (left.value.type) {
                    case M_T_BOOL:
                        if (!left.value.as.boolean)
                            return m_result_normal(m_value_bool(false));
                        break;
                    case M_T_INT:
                        if (left.value.as.integer == 0)
                            return m_result_normal(m_value_bool(false));
                        break;
                    case M_T_FLOAT:
                        if (left.value.as.floating == 0.0)
                            return m_result_normal(m_value_bool(false));
                        break;
                    default:
                        break;
                }

                M_Eval_Result right = m_result_expect_type(expression->Binary.right, evaluate_expression(interpreter, expression->Binary.right), M_T_BOOL | M_T_INT | M_T_FLOAT);

                switch (right.value.type) {
                    case M_T_BOOL:
                        if (!right.value.as.boolean)
                            return m_result_normal(m_value_bool(false));
                        break;
                    case M_T_INT:
                        if (right.value.as.integer == 0)
                            return m_result_normal(m_value_bool(false));
                        break;
                    case M_T_FLOAT:
                        if (right.value.as.floating == 0.0)
                            return m_result_normal(m_value_bool(false));
                        break;
                    default:
                        break;
                }

                return m_result_normal(m_value_bool(true));
            }
        case M_BINARY_OR_OP:
            {
                M_Eval_Result left = m_result_expect_type(expression->Binary.left, evaluate_expression(interpreter, expression->Binary.left), M_T_BOOL | M_T_INT | M_T_FLOAT);

                switch (left.value.type) {
                    case M_T_BOOL:
                        if (left.value.as.boolean)
                            return m_result_normal(m_value_bool(true));
                        break;
                    case M_T_INT:
                        if (left.value.as.integer != 0)
                            return m_result_normal(m_value_bool(true));
                        break;
                    case M_T_FLOAT:
                        if (left.value.as.floating != 0.0)
                            return m_result_normal(m_value_bool(true));
                        break;
                    default:
                        break;
                }

                M_Eval_Result right = m_result_expect_type(expression->Binary.right, evaluate_expression(interpreter, expression->Binary.right), M_T_BOOL | M_T_INT | M_T_FLOAT);

                switch (right.value.type) {
                    case M_T_BOOL:
                        if (right.value.as.boolean)
                            return m_result_normal(m_value_bool(true));
                        break;
                    case M_T_INT:
                        if (right.value.as.integer != 0)
                            return m_result_normal(m_value_bool(true));
                        break;
                    case M_T_FLOAT:
                        if (right.value.as.floating != 0.0)
                            return m_result_normal(m_value_bool(true));
                        break;
                    default:
                        break;
                }

                return m_result_normal(m_value_bool(false));
            }
        default:
            break;
    }

    M_Eval_Result left = m_result_expect_type(expression->Binary.left, evaluate_expression(interpreter, expression->Binary.left), M_T_INT | M_T_FLOAT | M_T_BOOL | M_T_UNIT | M_T_STRING);
    M_Eval_Result right = m_result_expect_type(expression->Binary.right, evaluate_expression(interpreter, expression->Binary.right), M_T_INT | M_T_FLOAT | M_T_BOOL | M_T_UNIT | M_T_STRING);

    M_Value_Type l_type = left.value.type;
    M_Value_Type r_type = right.value.type;

    if (l_type == M_T_UNIT || r_type == M_T_UNIT) {
        switch (expression->Binary.op) {
            case M_BINARY_EQUAL_OP:
                return m_result_normal(m_value_bool(l_type == r_type));
            case M_BINARY_NOT_EQUAL_OP:
                return m_result_normal(m_value_bool(l_type != r_type));
            default:
                m_interpreter_error(
                    l_type == M_T_UNIT ? expression->Binary.left : expression->Binary.right,
                    "invalid operation. cannot perform binary operation '%s' with unit type",
                    binary_expression_operator_name(expression->Binary.op)
                );
                break;
        }
    }

    if (l_type == M_T_STRING || l_type == M_T_STRING) {
        if (l_type != r_type) {
            m_interpreter_error(
                l_type != M_T_STRING ? expression->Binary.left : expression->Binary.right,
                "you cannot do binary operations between this types '%s %s %s'",
                m_value_type_name(l_type),
                binary_expression_operator_name(expression->Binary.op),
                m_value_type_name(r_type)
            );
        }

        static_assert(M_BINARY_OP_COUNT == 14, "evaluate_binary_expression: unhandled binary operator");
        switch (expression->Binary.op) {
            case M_BINARY_EQUAL_OP: {
                bool are_equal = left.value.as.string.value_length == right.value.as.string.value_length &&
                    strncmp(left.value.as.string.value, right.value.as.string.value, right.value.as.string.value_length) == 0;

                return m_result_normal(m_value_bool(are_equal));
            }
            case M_BINARY_NOT_EQUAL_OP: {
                bool arent_equal = left.value.as.string.value_length != right.value.as.string.value_length ||
                    strncmp(left.value.as.string.value, right.value.as.string.value, right.value.as.string.value_length) != 0;
                
                return m_result_normal(m_value_bool(arent_equal));
            }
            case M_BINARY_PLUS_OP:
            case M_BINARY_TIMES_OP:
            case M_BINARY_DIVIDE_OP:
            case M_BINARY_SUBTRACT_OP:
            case M_BINARY_MOD_OP:
            case M_BINARY_POW_OP:
            case M_BINARY_AND_OP:
            case M_BINARY_OR_OP:
            case M_BINARY_GT_OP:
            case M_BINARY_LT_OP:
            case M_BINARY_GTE_OP:
            case M_BINARY_LTE_OP:
                m_interpreter_error(
                    l_type != M_T_STRING ? expression->Binary.left : expression->Binary.right,
                    "you cannot do binary operation '%s' between strings",
                    binary_expression_operator_name(expression->Binary.op)
                );
                break;
            case M_BINARY_OP_COUNT: break;
        }
    }

    bool returns_bool = false;

    switch (expression->Binary.op) {
        case M_BINARY_AND_OP:
        case M_BINARY_OR_OP:
        case M_BINARY_EQUAL_OP:
        case M_BINARY_NOT_EQUAL_OP:
        case M_BINARY_GT_OP:
        case M_BINARY_LT_OP:
        case M_BINARY_GTE_OP:
        case M_BINARY_LTE_OP:
            returns_bool = true;
            break;
        default:
            break;
    }

    if (l_type == M_T_FLOAT || r_type == M_T_FLOAT) {
        double l_val = (l_type == M_T_FLOAT) ? left.value.as.floating : (l_type == M_T_BOOL) ? (double)left.value.as.boolean : (double)left.value.as.integer;
        double r_val = (r_type == M_T_FLOAT) ? right.value.as.floating : (r_type == M_T_BOOL) ? (double)right.value.as.boolean : (double)right.value.as.integer;
        double result = evaluate_binary_operation_on_doubles(expression->Binary.op, l_val, r_val);

        if (returns_bool) {
            return (M_Eval_Result){
                .value = {
                    .type = M_T_BOOL,
                    .as.boolean = result != 0.0
                },
                .flow = M_CTRL_NORMAL
            };
        }

        return (M_Eval_Result){
            .value = {
                .type = M_T_FLOAT,
                .as.floating = result
            },
            .flow = M_CTRL_NORMAL
        };
    }

    int64_t l = l_type == M_T_INT ? left.value.as.integer : (int)left.value.as.boolean;
    int64_t r = r_type == M_T_INT ? right.value.as.integer : (int)right.value.as.boolean;
    double result = 0;

    static_assert(M_BINARY_OP_COUNT == 14, "evaluate_binary_expression: unhandled binary operator");
    switch (expression->Binary.op) {
        case M_BINARY_PLUS_OP:      result = l + r; break;
        case M_BINARY_TIMES_OP:     result = l * r; break;
        case M_BINARY_DIVIDE_OP:    result = (double)l / (double)r; break; // TODO: handle division by zero
        case M_BINARY_SUBTRACT_OP:  result = l - r; break;
        case M_BINARY_MOD_OP:       result = l % r; break;
        case M_BINARY_POW_OP:       result = pow(l, r); break;
        case M_BINARY_EQUAL_OP:     result = l == r; break;
        case M_BINARY_NOT_EQUAL_OP: result = l != r; break;
        case M_BINARY_GT_OP:        result = l > r; break;
        case M_BINARY_LT_OP:        result = l < r; break;
        case M_BINARY_GTE_OP:       result = l >= r; break;
        case M_BINARY_LTE_OP:       result = l <= r; break;

        case M_BINARY_AND_OP: assert(0 && "evaluate_binary_expression: should never happen"); break; // it's being handled at the beginning of the function
        case M_BINARY_OR_OP: assert(0 && "evaluate_binary_expression: should never happen"); break;
        case M_BINARY_OP_COUNT: break;
    }

    if (returns_bool)
        return m_result_normal(m_value_bool(result != 0.0));

    if (fmod(result, 1.0) != 0.0)
        return m_result_normal(m_value_float(result));

    return m_result_normal(m_value_int((int64_t)result));
}

static M_Eval_Result evaluate_assignment_right_side(M_Interpreter *interpreter, M_Expression *expression) {
    M_Eval_Result right = evaluate_expression(interpreter, expression->Assign.right);

    if (expression->kind == M_EK_ASSIGN) return right;

    M_Eval_Result left_result = evaluate_expression(interpreter, expression->Assign.left);

    M_Value left = m_value_expect_type(expression->Assign.left, left_result.value, M_T_INT | M_T_FLOAT);

    M_Value_Type l_type = left.type;
    M_Value_Type r_type = right.value.type;

    if (l_type == M_T_FLOAT || r_type == M_T_FLOAT) {
        double l_value = l_type == M_T_FLOAT ? left.as.floating : (double)left.as.integer;
        double r_value = r_type == M_T_FLOAT ? right.value.as.floating : (double)right.value.as.integer;

        if (expression->kind == M_EK_ADD_ASSIGN) {
            return m_result_normal(m_value_float(l_value + r_value));
        } else if (expression->kind == M_EK_SUB_ASSIGN) {
            return m_result_normal(m_value_float(l_value - r_value));
        } else {
            assert(0 && "unreacheable");
        }
    } else {
        if (expression->kind == M_EK_ADD_ASSIGN) {
            return m_result_normal(m_value_int(left.as.integer + right.value.as.integer));
        } else if (expression->kind == M_EK_SUB_ASSIGN) {
            return m_result_normal(m_value_int(left.as.integer - right.value.as.integer));
        } else {
            assert(0 && "unreacheable");
        }
    }

    return m_result_normal(m_value_unit());
}

static size_t __get_map_key_data_and_size_helper(M_Value *value, void **out, bool keep_nullbyte) {
    static_assert((ACCEPTABLE_MAP_KEY_TYPES) == (M_T_INT | M_T_STRING), "ACCEPTABLE_MAP_KEY_TYPES has changed");

    switch (value->type) {
        case M_T_INT:
            *out = &value->as.integer;
            return sizeof(M_Int);
        case M_T_STRING:
            *out = value->as.string.value;
            return value->as.string.value_length + (keep_nullbyte ? 1 : 0);
        default:
            assert(0 && "should never happen");
            break;
    }

    return 0;
}

static size_t __get_map_value_data_and_size_helper(M_Value *value, void **out, bool keep_nullbyte) {
    static_assert((ACCEPTABLE_MAP_VALUE_TYPES) == (M_T_STRING | M_T_INT | M_T_BOOL | M_T_FLOAT | M_T_FN), "ACCEPTABLE_MAP_VALUE_TYPES has changed");

    switch (value->type) {
        case M_T_STRING:
            *out = value->as.string.value;
            return value->as.string.value_length + (keep_nullbyte ? 1 : 0);
        case M_T_INT:
            *out = &value->as.integer;
            return sizeof(M_Int);
        case M_T_BOOL:
            *out = &value->as.boolean;
            return sizeof(M_Bool);
        case M_T_FLOAT:
            *out = &value->as.floating;
            return sizeof(M_Float);
        case M_T_FN:
            *out = &value->as.fn;
            return sizeof(M_Expression*);
        case M_T_MAP:
        case M_T_MAP_IT:
        case M_T_ARRAY:
        case M_T_UNIT:
        case M_T_COUNT:
            assert(0 && "should never happen");
            break;
    }

    return 0;
}

static M_Value __builtin_mca_map_parse_m_map_node_entry_helper(M_Map_Node_Entry *entry) {
    if (entry == NULL) return m_value_unit();

    static_assert(M_T_COUNT == 257, "__builtin_mca_map_parse_m_map_node_entry_helper");
    switch (entry->type) {
        case M_T_STRING:
            return (M_Value){
                .allocated = true,
                .type = M_T_STRING,
                .as.string.value = entry->data,
                .as.string.value_length = entry->size - 1 // ignore the null-byte on counting length
            };
        case M_T_INT:
            return (M_Value){
                .type = M_T_INT,
                .as.integer = *(M_Int*)entry->data,
            };
        case M_T_BOOL:
            return (M_Value){
                .type = M_T_BOOL,
                .as.boolean = *(M_Bool*)entry->data,
            };
        case M_T_FLOAT:
            return (M_Value){
                .type = M_T_FLOAT,
                .as.floating = *(M_Float*)entry->data,
            };
        case M_T_FN:
            return (M_Value){
                .type = M_T_FN,
                .as.fn = *(M_Expression**)entry->data
            };
        case M_T_UNIT:
        case M_T_MAP_IT:
        case M_T_MAP:
            assert(0 && "TODO: for now, we don't accept nested hashmaps nor iterators or unit as value");
            break;
    }

    assert(0 && "should never happen");
}

static M_Eval_Result evaluate_assignment_expression(M_Interpreter *interpreter, M_Expression *expression) {
    M_Eval_Result right_side_value = evaluate_assignment_right_side(interpreter, expression);

    static_assert((ACCEPTABLE_ASSIGNMENT_LEFT_SIDE_EXPRESSION_KINDS) == (M_EK_ID | M_EK_INDEX), "ACCEPTABLE_ASSIGNMENT_LEFT_SIDE_EXPRESSION_KINDS has changed");

    if (expression->Assign.left->kind == M_EK_ID) {
        set_variable_on_environment(interpreter, interpreter->current_environment, expression->Assign.left->Id.value, right_side_value.value);

        return right_side_value;
    }

    if (expression->Assign.left->kind == M_EK_INDEX) {
        M_Eval_Result left = m_result_expect_type(expression->Assign.left->Index.left, evaluate_expression(interpreter, expression->Assign.left->Index.left), M_T_ARRAY | M_T_MAP);

        switch (left.value.type) {
            case M_T_MAP: {
                static_assert((ACCEPTABLE_MAP_KEY_TYPES) == (M_T_INT | M_T_STRING), "ACCEPTABLE_MAP_KEY_TYPES has changed");

                // ensure the correct type
                m_value_expect_type(expression->Assign.right, right_side_value.value, ACCEPTABLE_MAP_VALUE_TYPES);

                void *value = NULL;
                size_t value_size = __get_map_value_data_and_size_helper(&right_side_value.value, &value, true);

                if (expression->Assign.left->Index.index->kind == M_EK_ID) {
                    mca_map_set(left.value.as.map, expression->Assign.left->Index.index->Id.value, expression->Assign.left->Index.index->Id.value_length + 1, M_T_STRING, value, value_size, right_side_value.value.type);
                } else {
                    M_Eval_Result index = m_result_expect_type(expression->Assign.left->Index.index, evaluate_expression(interpreter, expression->Assign.left->Index.index), ACCEPTABLE_MAP_KEY_TYPES);

                    void *key = NULL;
                    size_t key_size = __get_map_key_data_and_size_helper(&index.value, &key, true);

                    mca_map_set(left.value.as.map, key, key_size, index.value.type, value, value_size, right_side_value.value.type);
                }

                return right_side_value;
            }
            case M_T_ARRAY: {
                static_assert((ACCEPTABLE_ARRAY_KEY_TYPES) == (M_T_INT), "ACCEPTABLE_ARRAY_KEY_TYPES has changed");

                M_Eval_Result index = m_result_expect_type(expression->Assign.left->Index.index, evaluate_expression(interpreter, expression->Assign.left->Index.index), ACCEPTABLE_ARRAY_KEY_TYPES);
                M_Int idx = index.value.as.integer;

                if (idx < 0 || idx >= left.value.as.array->length)
                    m_interpreter_error(expression, "array index out of bounds");

                left.value.as.array->items[idx] = right_side_value.value;

                return right_side_value;
            }
            default:
                assert(0 && "should never happen");
        }
    }

    return right_side_value;
}

static M_Eval_Result evaluate_while_loop_expression(M_Interpreter *interpreter, M_Expression *expression) {
    M_Eval_Result last_evaluated_expression = m_result_normal(m_value_unit());

    while (1) {
        if (expression->While.condition != NULL) {
            M_Eval_Result condition = evaluate_expression(interpreter, expression->While.condition);

            int evaluated_condition = evaluate_m_value_as_internal_boolean(condition.value);

            if (evaluated_condition == -1) {
                m_interpreter_error(expression->While.condition, "failed to check truthiness of '%s' data type on that 'loop'", m_value_type_name(condition.value.type));
            }

            if (!evaluated_condition) break;
        }


        if (expression->While.block != NULL) {
            // entering the loop block
            enter_new_environment(interpreter);

            last_evaluated_expression = evaluate_block_expression(interpreter, expression->While.block);

            // quiting the loop block
            destroy_current_environment(interpreter);

            if (last_evaluated_expression.flow == M_CTRL_RETURN) {
                return last_evaluated_expression;
            }

            if (last_evaluated_expression.flow == M_CTRL_BREAK) {
                last_evaluated_expression = m_result_normal(last_evaluated_expression.value);
                break;
            }
        }
    }

    return last_evaluated_expression;
}

static M_Eval_Result evaluate_for_of_loop_for_map_expression(M_Interpreter *interpreter, M_Expression *expression, M_Value target) {
    M_Eval_Result last_evaluated_expression = m_result_normal(m_value_unit());

    M_Map_Iterator *it = mca_map_iterator(target.as.map);

    while (!mca_map_iterator_finished(it)) {
        if (expression->ForOf.block != NULL) {
            enter_new_environment(interpreter);

            M_Value key   = __builtin_mca_map_parse_m_map_node_entry_helper(&it->it->key);
            M_Value value = __builtin_mca_map_parse_m_map_node_entry_helper(&it->it->value);

            define_variable_in_environment(interpreter->current_environment, expression->ForOf.key->Id.value, key);
            define_variable_in_environment(interpreter->current_environment, expression->ForOf.value->Id.value, value);

            last_evaluated_expression = evaluate_block_expression(interpreter, expression->ForOf.block);

            destroy_current_environment(interpreter);

            if (last_evaluated_expression.flow != M_CTRL_NORMAL) {
                mca_map_iterator_free(it);

                return last_evaluated_expression;
            }
        }

        mca_map_iterator_next(it);
    }

    return last_evaluated_expression;
}

static M_Eval_Result evaluate_for_of_loop_for_array_expression(M_Interpreter *interpreter, M_Expression *expression, M_Value target) {
    M_Eval_Result last_evaluated_expression = m_result_normal(m_value_unit());

    for (int i = 0; i < target.as.array->length; i++) {
        if (expression->ForOf.block != NULL) {
            enter_new_environment(interpreter);

            M_Value key = { .as.integer = i,.type = M_T_INT };
            M_Value value = target.as.array->items[i];

            define_variable_in_environment(interpreter->current_environment, expression->ForOf.key->Id.value, key);
            define_variable_in_environment(interpreter->current_environment, expression->ForOf.value->Id.value, value);

            last_evaluated_expression = evaluate_block_expression(interpreter, expression->ForOf.block);

            destroy_current_environment(interpreter);

            if (last_evaluated_expression.flow != M_CTRL_NORMAL) {
                return last_evaluated_expression;
            }
        }
    }

    return last_evaluated_expression;
}

static M_Eval_Result evaluate_for_of_loop_for_string_expression(M_Interpreter *interpreter, M_Expression *expression, M_Value target) {
    M_Eval_Result last_evaluated_expression = m_result_normal(m_value_unit());

    for (int i = 0; i < target.as.string.value_length; i++) {
        if (expression->ForOf.block != NULL) {
            enter_new_environment(interpreter);

            M_Value key = { .as.integer = i, .type = M_T_INT };
            M_Value value = { .as.string.value = target.as.string.value + i, .as.string.value_length = 1, .allocated = false, .type = M_T_STRING };

            define_variable_in_environment(interpreter->current_environment, expression->ForOf.key->Id.value, key);
            define_variable_in_environment(interpreter->current_environment, expression->ForOf.value->Id.value, value);

            last_evaluated_expression = evaluate_block_expression(interpreter, expression->ForOf.block);

            destroy_current_environment(interpreter);

            if (last_evaluated_expression.flow != M_CTRL_NORMAL) {
                return last_evaluated_expression;
            }
        }
    }

    return last_evaluated_expression;
}

static M_Eval_Result evaluate_for_of_loop_expression(M_Interpreter *interpreter, M_Expression *expression) {
    static_assert((ACCEPTABLE_FOR_OF_TARGET_VALUE_TYPES) == (M_T_ARRAY | M_T_STRING | M_T_MAP), "ACCEPTABLE_FOR_OF_TARGET_VALUE_TYPES has changed");

    M_Eval_Result target = m_result_expect_type(expression->ForOf.target, evaluate_expression(interpreter, expression->ForOf.target), ACCEPTABLE_FOR_OF_TARGET_VALUE_TYPES);

    switch (target.value.type) {
        case M_T_MAP: return evaluate_for_of_loop_for_map_expression(interpreter, expression, target.value);
        case M_T_ARRAY: return evaluate_for_of_loop_for_array_expression(interpreter, expression, target.value);
        case M_T_STRING: return evaluate_for_of_loop_for_string_expression(interpreter, expression, target.value);
        default: assert(0 && "should never happen");
    }
}

static M_Eval_Result evaluate_for_range_loop_expression_base(M_Interpreter *interpreter, M_Expression *expression, M_Int from, M_Int to, M_Int by) {
    M_Eval_Result last_evaluated_expression = m_result_normal(m_value_unit());

    int is_negative = from * to < 0 ? 1 : 0;

    for (int i = from; is_negative ? i > to : i < to; i += by) {
        if (expression->ForRange.block != NULL) {
            enter_new_environment(interpreter);

            M_Value key = { .as.integer = i, .type = M_T_INT };

            define_variable_in_environment(interpreter->current_environment, expression->ForRange.index->Id.value, key);

            last_evaluated_expression = evaluate_block_expression(interpreter, expression->ForRange.block);

            destroy_current_environment(interpreter);

            if (last_evaluated_expression.flow != M_CTRL_NORMAL) {
                return last_evaluated_expression;
            }
        }
    }

    return last_evaluated_expression;
}


static M_Eval_Result evaluate_simple_for_range_loop(M_Interpreter *interpreter, M_Expression *expression, M_Value from) {
    return evaluate_for_range_loop_expression_base(interpreter, expression, 0, from.as.integer, 1);
}

static M_Eval_Result evaluate_default_for_range_loop(M_Interpreter *interpreter, M_Expression *expression, M_Value from) {
    M_Value to = m_result_expect_type(expression->ForRange.to, evaluate_expression(interpreter, expression->ForRange.to), M_T_INT).value;

    return evaluate_for_range_loop_expression_base(interpreter, expression, from.as.integer, to.as.integer, 1);
}

static M_Eval_Result evaluate_complex_for_range_loop(M_Interpreter *interpreter, M_Expression *expression, M_Value from) {
    M_Value to = m_result_expect_type(expression->ForRange.to, evaluate_expression(interpreter, expression->ForRange.to), M_T_INT).value;
    M_Value by = m_result_expect_type(expression->ForRange.by, evaluate_expression(interpreter, expression->ForRange.by), M_T_INT).value;

    return evaluate_for_range_loop_expression_base(interpreter, expression, from.as.integer, to.as.integer, by.as.integer);
}

static M_Eval_Result evaluate_for_range_loop_expression(M_Interpreter *interpreter, M_Expression *expression) {
    M_Eval_Result last_evaluated_expression = m_result_normal(m_value_unit());

    M_Value from = m_result_expect_type(expression->ForRange.from, evaluate_expression(interpreter, expression->ForRange.from), M_T_INT).value;

    if (expression->ForRange.to == NULL) return evaluate_simple_for_range_loop(interpreter, expression, from);
    if (expression->ForRange.by == NULL) return evaluate_default_for_range_loop(interpreter, expression, from);
    else                                 return evaluate_complex_for_range_loop(interpreter, expression, from);

    return last_evaluated_expression;
}

static M_Eval_Result evaluate_expression(M_Interpreter *interpreter, M_Expression *expression) {
    assert(expression != NULL && "evaluate_expression_impl: expression cannot be null");

    switch (expression->kind) {
        case M_EK_STRING: return m_result_normal(m_value_string(expression->String));
        case M_EK_UNIT:   return m_result_normal(m_value_unit());
        case M_EK_BOOL:   return m_result_normal(m_value_bool(expression->Bool));
        case M_EK_INT:    return m_result_normal(m_value_int(expression->Int));
        case M_EK_FLOAT:  return m_result_normal(m_value_float(expression->Float));
        case M_EK_ID: {
            char *key = strndup(expression->Id.value, expression->Id.value_length);

            M_Value *value = get_variable_from_environment(interpreter->current_environment, key);

            if (value == NULL)
                m_interpreter_error(expression, "variable '%s' does not exists", key);

            free(key);

            return m_result_normal(*value);
        };
        case M_EK_ASSIGN:     return evaluate_assignment_expression(interpreter, expression);
        case M_EK_ADD_ASSIGN: return evaluate_assignment_expression(interpreter, expression);
        case M_EK_SUB_ASSIGN: return evaluate_assignment_expression(interpreter, expression);
        case M_EK_UNARY: {
            switch (expression->Unary.op) {
                case M_UNARY_MINUS_OP: {
                    M_Eval_Result result = evaluate_expression(interpreter, expression->Unary.operand);
                    M_Value_Union out = {0};

                    switch (result.value.type) {
                        case M_T_INT:
                            out.integer = -result.value.as.integer;
                            break;
                        case M_T_FLOAT:
                            out.floating = -result.value.as.floating;
                            break;
                        default:
                            m_interpreter_error(expression, "unexpected data type. expected a int or float but got a '%s'", m_value_type_name(result.value.type));
                            break;
                    }

                    return (M_Eval_Result){
                        .flow = result.flow,
                        .value = (M_Value){
                            .type = result.value.type,
                            .as = out
                        }
                    };
                } 
                case M_UNARY_NOT_OP: {
                    M_Eval_Result result = m_result_expect_type(expression, evaluate_expression(interpreter, expression->Unary.operand), M_T_INT | M_T_FLOAT | M_T_BOOL);

                    switch (result.value.type) {
                        case M_T_BOOL:
                            return (M_Eval_Result){
                                .flow = result.flow,
                                .value = m_value_bool(!result.value.as.boolean)
                            };
                        case M_T_INT:
                            return (M_Eval_Result){
                                .flow = result.flow,
                                .value = m_value_bool(!result.value.as.integer)
                            };
                        case M_T_FLOAT:
                            return (M_Eval_Result){
                                .flow = result.flow,
                                .value = m_value_bool(!result.value.as.floating)
                            };
                        case M_T_STRING:
                        case M_T_UNIT:
                        case M_T_ARRAY:
                        case M_T_MAP:
                        case M_T_MAP_IT:
                        case M_T_COUNT:
                        case M_T_FN:
                            assert(0 && "case M_UNARY_NOT_OP: unreachable");
                            break;
                    }
                } break;
                case M_UNARY_FACTORIAL_OP: return calculate_factorial(m_result_expect_type(expression, evaluate_expression(interpreter, expression->Unary.operand), M_T_INT | M_T_FLOAT));
            }

            assert(0 && "evaluate_expression_impl: invalid unary expression operator");
        } break;

        case M_EK_ARRAY: {
            M_Array *array = malloc(sizeof(M_Array));
            array->length = expression->Array.items_length;
            array->capacity = array->length == 0 ? 8 : array->length;
            array->items = malloc(sizeof(M_Value) * array->capacity);

            for (int i = 0; i < array->length; i++) {
                array->items[i] = evaluate_expression(interpreter, expression->Array.items[i]).value;
            }

            M_Value val = (M_Value){.allocated = true, .type = M_T_ARRAY, .as = {.array = array}};
            return m_result_normal(val);
        }
        case M_EK_INDEX: {
            M_Eval_Result left = m_result_expect_type(expression->Index.left, evaluate_expression(interpreter, expression->Index.left), M_T_ARRAY | M_T_STRING | M_T_MAP);

            switch (left.value.type) {
                case M_T_ARRAY: {
                    static_assert((ACCEPTABLE_ARRAY_KEY_TYPES) == (M_T_INT), "ACCEPTABLE_ARRAY_KEY_TYPES has changed");
                    M_Eval_Result index = m_result_expect_type(expression->Index.index, evaluate_expression(interpreter, expression->Index.index), ACCEPTABLE_ARRAY_KEY_TYPES);
                    M_Int idx = m_value_expect_type(expression->Index.index, index.value, M_T_INT).as.integer;

                    if (idx < 0 || idx >= left.value.as.array->length)
                        m_interpreter_error(expression, "array index out of bounds");

                    return m_result_normal(left.value.as.array->items[idx]);
                };
                case M_T_STRING: {
                    static_assert((ACCEPTABLE_ARRAY_KEY_TYPES) == (M_T_INT), "ACCEPTABLE_ARRAY_KEY_TYPES has changed");
                    M_Eval_Result index = m_result_expect_type(expression->Index.index, evaluate_expression(interpreter, expression->Index.index), ACCEPTABLE_ARRAY_KEY_TYPES);
                    M_Int idx = m_value_expect_type(expression->Index.index, index.value, M_T_INT).as.integer;

                    if (idx < 0 || idx >= left.value.as.string.value_length)
                        m_interpreter_error(expression, "string index out of bounds");
                    return m_result_normal(m_value_sized_string(left.value.as.string.value + idx, 1));
                }
                case M_T_MAP: {
                    static_assert((ACCEPTABLE_MAP_KEY_TYPES) == (M_T_INT | M_T_STRING), "ACCEPTABLE_MAP_KEY_TYPES has changed");

                    if (expression->Index.index->kind == M_EK_CALL) {
                        M_Map_Node_Entry *entry = mca_map_find(left.value.as.map, expression->Index.index->Call.fn_name.value, expression->Index.index->Call.fn_name.value_length + 1, M_T_STRING);

                        M_Value value = m_value_expect_type(expression->Index.index, __builtin_mca_map_parse_m_map_node_entry_helper(entry), M_T_FN);

                        M_Value call_value = evaluate_function_execution(interpreter, value.as.fn, expression->Index.index);

                        return m_result_normal(call_value);
                    } else if (expression->Index.index->kind == M_EK_ID) {
                        M_Map_Node_Entry *entry = mca_map_find(left.value.as.map, expression->Index.index->Id.value, expression->Index.index->Id.value_length + 1, M_T_STRING);

                        M_Value value = __builtin_mca_map_parse_m_map_node_entry_helper(entry);

                        return m_result_normal(value);
                    } else {
                        M_Eval_Result index = m_result_expect_type(expression->Index.index, evaluate_expression(interpreter, expression->Index.index), ACCEPTABLE_MAP_KEY_TYPES);
                        M_Value idx = m_value_expect_type(expression->Index.index, index.value, ACCEPTABLE_MAP_KEY_TYPES);

                        void *key = NULL;
                        size_t key_size = __get_map_key_data_and_size_helper(&idx, &key, true);

                        M_Map_Node_Entry *entry = mca_map_find(left.value.as.map, key, key_size, idx.type);

                        M_Value value = __builtin_mca_map_parse_m_map_node_entry_helper(entry);

                        return m_result_normal(value);
                    }
                };
                default: {
                    static_assert(M_T_COUNT == 257, "handle index type");
                    break;
                }
            }

            assert(0 && "should never happen");
            break;
        }
        case M_EK_CALL: return m_result_normal(evaluate_function_call_expression(interpreter, expression));
        case M_EK_BINARY: return evaluate_binary_expression(interpreter, expression);
        case M_EK_IF: {
            M_Eval_Result condition = evaluate_expression(interpreter, expression->If.condition);
            M_Eval_Result last_evaluated_expression = m_result_normal(m_value_unit());

            int evaluated_condition = evaluate_m_value_as_internal_boolean(condition.value);

            if (evaluated_condition == -1)
                m_interpreter_error(expression->If.condition, "failed to check truthiness of '%s' data type on that 'if'", m_value_type_name(condition.value.type));

            if (evaluated_condition) {
                enter_new_environment(interpreter); // enter 'if' block
                last_evaluated_expression = evaluate_block_expression(interpreter, expression->If.then_block);
                destroy_current_environment(interpreter); // quit 'if' block
            } else {
                if (expression->If.elif_blocks != NULL) {
                    M_Expression_Elif_Block *current_elif = expression->If.elif_blocks;

                    while (current_elif != NULL) {
                        M_Eval_Result elif_condition = evaluate_expression(interpreter, current_elif->condition);

                        int evaluated_elif_condition = evaluate_m_value_as_internal_boolean(elif_condition.value);

                        if (evaluated_elif_condition == -1)
                            m_interpreter_error(current_elif->condition, "failed to check truthiness of '%s' data type on that 'elif'", m_value_type_name(elif_condition.value.type));

                        if (evaluated_elif_condition) {
                            if (current_elif->block != NULL) {
                                enter_new_environment(interpreter); // enter 'elif' block
                                last_evaluated_expression = evaluate_block_expression(interpreter, current_elif->block);
                                destroy_current_environment(interpreter); // quit 'elif' block
                            }

                            return last_evaluated_expression;
                        }

                        current_elif = current_elif->next;
                    }
                }

                if (expression->If.else_block != NULL) {
                    enter_new_environment(interpreter); // enter 'else' block
                    last_evaluated_expression = evaluate_block_expression(interpreter, expression->If.else_block);
                    destroy_current_environment(interpreter); // quit 'else' block
                }
            }

            return last_evaluated_expression;
        } break;
        case M_EK_FOR_RANGE: return evaluate_for_range_loop_expression(interpreter, expression);
        case M_EK_FOR_OF: return evaluate_for_of_loop_expression(interpreter, expression);
        case M_EK_WHILE: return evaluate_while_loop_expression(interpreter, expression);
        case M_EK_BREAK: {
            if (expression->Break != NULL) {
                M_Eval_Result result = evaluate_expression(interpreter, expression->Break);

                return m_result_break(result.value);
            }

            return m_result_break(m_value_unit());
        };
        case M_EK_RETURN: {
            if (expression->Return != NULL) {
                M_Eval_Result result = evaluate_expression(interpreter, expression->Return);

                return m_result_return(result.value);
            }

            return m_result_return(m_value_unit());
        };
        case M_EK_FN:
            expression->Fn.closure_env = interpreter->current_environment;
            acquire_environment(expression->Fn.closure_env);
            return m_result_normal(m_value_fn(expression));
        case M_EK_MAP: {
            static_assert((ACCEPTABLE_MAP_KEY_TYPES) == (M_T_INT | M_T_STRING), "ACCEPTABLE_MAP_KEY_TYPES has changed");
            static_assert((ACCEPTABLE_MAP_VALUE_TYPES) == (M_T_STRING | M_T_INT | M_T_BOOL | M_T_FLOAT | M_T_FN), "ACCEPTABLE_MAP_VALUE_TYPES has changed");

            M_Map *map = mca_map_init();

            for (int i = 0; i < expression->Map.items_length; i += 2) {
                M_Expression *key_expr = expression->Map.items[i];
                M_Expression *value_expr = expression->Map.items[i+1];

                M_Eval_Result key_result = m_result_expect_type(key_expr, evaluate_expression(interpreter, key_expr), ACCEPTABLE_MAP_KEY_TYPES);
                M_Eval_Result value_result = m_result_expect_type(value_expr, evaluate_expression(interpreter, value_expr), ACCEPTABLE_MAP_VALUE_TYPES);

                void *key = NULL;
                size_t key_size = __get_map_key_data_and_size_helper(&key_result.value, &key, true);

                void *value = NULL;
                size_t value_size = __get_map_value_data_and_size_helper(&value_result.value, &value, true);

                mca_map_set(map, key, key_size, key_result.value.type, value, value_size, value_result.value.type);
            }

            M_Value value = (M_Value){
                .allocated = true,
                .type = M_T_MAP,
                .as.map = map
            };

            return m_result_normal(value);
        }
    }

    assert(0 && "should never happen");
}

PUBLIC M_Interpreter *m_interpreter_create(M_Ast *program, int argc, const char **argv) {
    M_Interpreter *interpreter = malloc(sizeof(M_Interpreter));

    interpreter->program = program;
    interpreter->io_in = stdin;
    interpreter->io_out = stdout;
    interpreter->io_err = stderr;

    interpreter->argc = argc;
    interpreter->argv = argv;

    interpreter->global_environment = malloc(sizeof(M_Environment));
    interpreter->global_environment->variables = ht_init(sizeof(M_Value));
    interpreter->global_environment->parent = NULL;
    interpreter->global_environment->ref_count = 1;
    interpreter->current_environment = interpreter->global_environment;

    return interpreter;
}

PUBLIC void m_interpreter_set_stdin(M_Interpreter *interpreter, FILE *stream) {
    interpreter->io_in = stream;
}

PUBLIC void m_interpreter_set_stdout(M_Interpreter *interpreter, FILE *stream) {
    interpreter->io_out = stream;
}

PUBLIC void m_interpreter_set_stderr(M_Interpreter *interpreter, FILE *stream) {
    interpreter->io_err = stream;
}

PUBLIC M_Value m_interpreter_run(M_Interpreter *interpreter) {
    if (interpreter->program == NULL) return m_value_unit();

    M_Value last_evaluated_expression = m_value_unit();

    for (int i = 0; i < interpreter->program->expressions_array_length; i++) {
        M_Expression *expr = interpreter->program->expressions_array[i];

        if (expr != NULL) {
            M_Eval_Result r = evaluate_expression(interpreter, expr);

            if (r.flow == M_CTRL_BREAK) m_interpreter_error(expr, "cannot use 'break' outside of a loop");
            if (r.flow == M_CTRL_RETURN) m_interpreter_error(expr, "cannot use 'return' outside of a function");

            last_evaluated_expression = r.value;
        }
    }

    return last_evaluated_expression;
}

PUBLIC void m_interpreter_free(M_Interpreter *interpreter) {
    ht_free(interpreter->global_environment->variables);
    free(interpreter->global_environment);
    ast_free(interpreter->program);
    free(interpreter);

    interpreter = NULL;
}

// HELPERS ----------------------------------------------------------------------------------------------------


static void __print_map_helper(M_Interpreter *interpreter, M_Map *map);
static void __print_value_helper(M_Interpreter *interpreter, M_Value value, bool wrap_strings);

static void __print_array_helper(M_Interpreter *interpreter, M_Array *array) {
    fprintf(interpreter->io_out, "[");
    for (int i = 0; i < array->length; i++) {
        if (i > 0) fprintf(interpreter->io_out, ", ");
        __print_value_helper(interpreter, array->items[i], true);
    }
    fprintf(interpreter->io_out, "]");
}

static void __print_value_helper(M_Interpreter *interpreter, M_Value value, bool wrap_strings) {
    switch (value.type) {
        case M_T_INT:
            fprintf(interpreter->io_out, "%ld", value.as.integer);
            break;
        case M_T_FLOAT:
            fprintf(interpreter->io_out, "%f", value.as.floating);
            break;
        case M_T_BOOL:
            fprintf(interpreter->io_out, "%s", value.as.boolean ? "true" : "false");
            break;
        case M_T_UNIT:
            fprintf(interpreter->io_out, "(unit)");
            break;
        case M_T_STRING:
            if (wrap_strings)
                fprintf(interpreter->io_out, "'%.*s'", value.as.string.value_length, value.as.string.value);
            else
                fprintf(interpreter->io_out, "%.*s", value.as.string.value_length, value.as.string.value);
            break;
        case M_T_MAP:
            __print_map_helper(interpreter, value.as.map);
            break;
        case M_T_MAP_IT:
            fprintf(interpreter->io_out, "*"); 
            __print_map_helper(interpreter, value.as.map_it->map);
            break;
        case M_T_FN:
            fprintf(interpreter->io_out, "fn(...%d)", value.as.fn->Fn.arguments_length);
            break;
        case M_T_ARRAY:
            __print_array_helper(interpreter, value.as.array);
            break;
        case M_T_COUNT:
            assert(0 && "__builtin_mca_print: unreachable M_T_COUNT");
            break;
    }
}

static void __print_map_helper(M_Interpreter *interpreter, M_Map *map) {
    fprintf(interpreter->io_out, "{");

    M_Map_Iterator *it = mca_map_iterator(map);
    
    for (int i = 0; !mca_map_iterator_finished(it); i++) {
        M_Value key = __builtin_mca_map_parse_m_map_node_entry_helper(&it->it->key);
        M_Value value = __builtin_mca_map_parse_m_map_node_entry_helper(&it->it->value);

        if (i > 0) fprintf(interpreter->io_out, ", ");

        __print_value_helper(interpreter, key, true);
        fprintf(interpreter->io_out, ": ");
        __print_value_helper(interpreter, value, true);

        mca_map_iterator_next(it);
    }

    mca_map_iterator_free(it);

    fprintf(interpreter->io_out, "}");
}

// BUILTIN FUNCTION IMPLEMENTATIONS ----------------------------------------------------------------------------------------------------

typedef struct {
    const char    *name;
    int            name_length;
    int            arguments_count;
    M_Fn_C_Impl    c_impl;
} M_Fn_Binding;

#define BIND_FN(fn_name, args, impl) { .name = fn_name, .name_length = sizeof(fn_name) - 1, .arguments_count = args, .c_impl = &impl }

#define DEFINE_MATH_BUILTIN(func_name, c_math_function) \
    static M_Value __builtin_mca_##func_name(M_Interpreter *interpreter, M_Expression *caller, M_Expression *arguments[], int arguments_count) { \
        (void)caller; \
        (void)arguments_count; \
        \
        M_Eval_Result arg = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT | M_T_FLOAT | M_T_BOOL); \
        \
        double input_val = (arg.value.type == M_T_FLOAT) \
                         ? arg.value.as.floating \
                         : arg.value.type == M_T_BOOL \
                           ? (double)arg.value.as.boolean \
                           : (double)arg.value.as.integer; \
        \
        double result = c_math_function(input_val); \
        \
        if (fmod(result, 1.0) != 0.0) { \
            return m_value_float(result); \
        } else { \
            return m_value_int((int64_t)result); \
        } \
    }

#define DEFINE_IS_TYPE_BUILTIN(name, dtype) \
static M_Value __builtin_mca_is_##name(M_Interpreter *interpreter, M_Expression *caller, M_Expression *arguments[], int arguments_count) { \
    (void)caller; \
    (void)arguments_count; \
    M_Eval_Result result = evaluate_expression(interpreter, arguments[0]); \
    return m_value_bool(result.value.type == dtype); \
}

static inline double calc_rad(double degrees) {
    return degrees * (M_PI / 180.0);
}

static inline double calc_deg(double radians) {
    return radians * (180.0 / M_PI);
}

DEFINE_MATH_BUILTIN(acos, acos)
DEFINE_MATH_BUILTIN(asin, asin)
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

DEFINE_IS_TYPE_BUILTIN(unit, M_T_UNIT)
DEFINE_IS_TYPE_BUILTIN(int, M_T_INT)
DEFINE_IS_TYPE_BUILTIN(float, M_T_FLOAT)
DEFINE_IS_TYPE_BUILTIN(string, M_T_STRING)
DEFINE_IS_TYPE_BUILTIN(bool, M_T_BOOL)

BUILTIN(__builtin_mca_pi) {
    (void)interpreter;
    (void)caller;
    (void)arguments;
    (void)arguments_count;

    return m_value_float(M_PI);
}

BUILTIN(__builtin_mca_e) {
    (void)interpreter;
    (void)caller;
    (void)arguments;
    (void)arguments_count;

    return m_value_float(M_E);
}

BUILTIN(__builtin_mca_abs) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result arg = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT | M_T_FLOAT | M_T_BOOL);

    if (arg.value.type == M_T_INT || arg.value.type == M_T_BOOL) {
        return m_value_int(llabs(arg.value.type == M_T_INT ? arg.value.as.integer : (int)arg.value.as.boolean));
    } else {
        return m_value_float(fabs(arg.value.as.floating));
    }
}

BUILTIN(__builtin_mca_max) {
    if (arguments_count < 1)
        m_interpreter_error(caller, "this function expects at least one argument");

    M_Eval_Result x = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT | M_T_FLOAT | M_T_BOOL);

    for (int i = 1; i < arguments_count; i++) {
        M_Eval_Result y = m_result_expect_type(arguments[i], evaluate_expression(interpreter, arguments[i]), M_T_INT | M_T_FLOAT | M_T_BOOL);

        if (x.value.type == M_T_INT && y.value.type == M_T_INT) {
            if (y.value.as.integer > x.value.as.integer) {
                x = y;
            }
        } else {
            double a0 = (x.value.type == M_T_FLOAT) ? x.value.as.floating : (double)x.value.as.integer;
            double a1 = (y.value.type == M_T_FLOAT) ? y.value.as.floating : (double)y.value.as.integer;

            if (a1 > a0) x = y;
        }
    }
    
    return x.value;
}

BUILTIN(__builtin_mca_min) {
    if (arguments_count < 1)
        m_interpreter_error(caller, "this function expects at least one argument");

    M_Eval_Result x = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT | M_T_FLOAT | M_T_BOOL);

    for (int i = 1; i < arguments_count; i++) {
        M_Eval_Result y = m_result_expect_type(arguments[i], evaluate_expression(interpreter, arguments[i]), M_T_INT | M_T_FLOAT | M_T_BOOL);

        if (x.value.type == M_T_INT && y.value.type == M_T_INT) {
            if (y.value.as.integer < x.value.as.integer) x = y;
        } else {
            double a0 = (x.value.type == M_T_FLOAT) ? x.value.as.floating : (double)x.value.as.integer;
            double a1 = (y.value.type == M_T_FLOAT) ? y.value.as.floating : (double)y.value.as.integer;

            if (a1 < a0) x = y;
        }
    }
    
    return x.value;
}

BUILTIN(__builtin_mca_print) {
    (void)caller;
    M_Value last_value = m_value_unit();

    for (int i = 0; i < arguments_count; i++) {
        last_value = evaluate_expression(interpreter, arguments[i]).value;

        __print_value_helper(interpreter, last_value, false);
    }

    return last_value;
}

BUILTIN(__builtin_mca_println) {
    (void)caller;
    M_Value last_value = m_value_unit();

    for (int i = 0; i < arguments_count; i++) {
        if (i > 0) fprintf(interpreter->io_out, " ");

        last_value = evaluate_expression(interpreter, arguments[i]).value;

        __print_value_helper(interpreter, last_value, false);
    }

    fprintf(interpreter->io_out, "\n");

    return last_value;
}

BUILTIN(__builtin_mca_exit) {
    (void)caller;
    (void)arguments_count;

    exit((int)m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT).value.as.integer);
}

BUILTIN(__builtin_mca_read_entire_file) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result a0 = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_STRING);

    const char *filename = a0.value.as.string.value;

    char *output; // null-terminated

    int size = read_entire_file_builtin(filename, &output);

    // TODO: how are going to deal with error handling?
    switch (size) {
        case -1:
            m_interpreter_error(caller, "coult not open file '%s' due to: '%s'", filename, strerror(errno));
            break;
        case -2:
            m_interpreter_error(caller, "could not allocate memory enough to read file %s due to: %s", filename, strerror(errno));
            break;
        case -3:
            m_interpreter_error(caller, "could not read data from file '%s' due to: %s", filename, strerror(errno));
            break;
        default:
            break;
    }

    return (M_Value){
        .type = M_T_STRING,
        .allocated = true,
        .as.string.value = output,
        .as.string.value_length = size
    };
}

BUILTIN(__builtin_mca_time) {
    (void)interpreter;
    (void)caller;
    (void)arguments;
    (void)arguments_count;

    return m_value_int(time(NULL));
}

BUILTIN(__builtin_mca_year) {
    (void)caller;
    (void)arguments_count;

    int64_t offset = (int)m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return m_value_int(time_info->tm_year + 1900);
}

BUILTIN(__builtin_mca_month) {
    (void)caller;
    (void)arguments_count;

    int offset = (int)m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return m_value_int(time_info->tm_mon + 1);
}

BUILTIN(__builtin_mca_date) {
    (void)caller;
    (void)arguments_count;

    int offset = (int)m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return m_value_int(time_info->tm_mday);
}

BUILTIN(__builtin_mca_day) {
    (void)caller;
    (void)arguments_count;

    int offset = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return m_value_int(time_info->tm_wday);
}

BUILTIN(__builtin_mca_hour) {
    (void)caller;
    (void)arguments_count;

    int offset = (int)m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return m_value_int(time_info->tm_hour);
}

BUILTIN(__builtin_mca_minute) {
    (void)caller;
    (void)arguments_count;

    int offset = (int)m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return m_value_int(time_info->tm_min);
}

BUILTIN(__builtin_mca_second) {
    (void)caller;
    (void)arguments_count;

    int offset = (int)m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT).value.as.integer;

    time_t current_time = time(NULL);
    time_t adjusted_time = current_time + (offset * 3600);

    struct tm *time_info = gmtime(&adjusted_time);

    return m_value_int(time_info->tm_sec);
}

BUILTIN(__builtin_mca_millisecond) {
    (void)interpreter;
    (void)caller;
    (void)arguments;
    (void)arguments_count;

    struct timespec ts;

    if (timespec_get(&ts, TIME_UTC) == -1) {
        m_interpreter_error(caller, "failed to get current time in milliseconds");
    }

    int64_t milliseconds = (int64_t)ts.tv_sec * 1000 + (ts.tv_nsec / 1000000);

    return m_value_int(milliseconds);
}

BUILTIN(__builtin_mca_type) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result result = evaluate_expression(interpreter, arguments[0]);

    switch (result.value.type) {
        case M_T_INT:
            return m_value_string(m_string("int"));
        case M_T_FLOAT:
            return m_value_string(m_string("float"));
        case M_T_BOOL:
            return m_value_string(m_string("bool"));
        case M_T_UNIT:
            return m_value_string(m_string("unit"));
        case M_T_STRING:
            return m_value_string(m_string("string"));
        case M_T_MAP:
            return m_value_string(m_string("map"));
        case M_T_ARRAY:
            return m_value_string(m_string("array"));
        case M_T_MAP_IT:
            return m_value_string(m_string("iter<map>"));
        case M_T_FN:
            return m_value_string(m_string("fn"));
        case M_T_COUNT:
            assert(0 && "__builtin_mca_type: unreachable M_T_COUNT");
            break;
    }

    assert(0 && "should never happen");
}

BUILTIN(__builtin_mca_as_int) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result result = evaluate_expression(interpreter, arguments[0]);

    switch (result.value.type) {
        case M_T_INT:   return result.value;
        case M_T_FLOAT: return m_value_int((int64_t)result.value.as.floating);
        case M_T_BOOL:  return m_value_int(result.value.as.boolean ? 1 : 0);
        case M_T_STRING: {
            char *endptr;
            int size = result.value.as.string.value_length;
            char *str = result.value.allocated ? result.value.as.string.value : strndup(result.value.as.string.value, result.value.as.string.value_length);

            errno = 0;

            int64_t v = strtoll(str, &endptr, 10);

            if (errno == ERANGE) {
                m_interpreter_error(arguments[0], "the number is too large or too small to fit in an integer type");
            } else if (endptr == str) {
                m_interpreter_error(arguments[0], "'%.*s' is not a valid number", size, str);
            } else if (*endptr != '\0') {
                m_interpreter_error(arguments[0], "'%.*s' is not a valid integer literal", size, str);
            }

            // free current allocation
            if (!result.value.allocated) free(str);

            return m_value_int(v);
        };
        default: m_interpreter_error(arguments[0], "cannot cast '%s' to int", m_value_type_name(result.value.type));
    }

    return result.value;
}

BUILTIN(__builtin_mca_as_float) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result result = evaluate_expression(interpreter, arguments[0]);

    switch (result.value.type) {
        case M_T_INT:   return m_value_float((double)result.value.as.integer);
        case M_T_FLOAT: return result.value;
        case M_T_BOOL:  return m_value_float(result.value.as.boolean ? 1.0 : 0.0);
        case M_T_STRING: {
            char *endptr = NULL;
            int size = result.value.as.string.value_length;
            char *str = result.value.as.string.value;

            errno = 0;

            double v = strtod(str, &endptr);

            if (errno == ERANGE) {
                m_interpreter_error(arguments[0], "the number is too large or too small to fit in a float type");
            } else if (endptr == str) {
                m_interpreter_error(arguments[0], "'%.*s' is not a valid number", size, str);
            } else if (*endptr != '\0') {
                m_interpreter_error(arguments[0], "'%.*s' is not a valid float literal: %s", size, str, endptr);
            }

            return m_value_float(v);
        };
        default: m_interpreter_error(arguments[0], "cannot cast '%s' to float", m_value_type_name(result.value.type));
    }

    return result.value;
}

BUILTIN(__builtin_mca_as_bool) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result result = evaluate_expression(interpreter, arguments[0]);

    switch (result.value.type) {
        case M_T_INT:   return m_value_bool(result.value.as.integer != 0);
        case M_T_FLOAT: return m_value_bool(result.value.as.floating != 0.0);
        case M_T_BOOL:  return result.value;
        default: m_interpreter_error(arguments[0], "cannot cast '%s' to bool", m_value_type_name(result.value.type));
    }

    return result.value;
}

BUILTIN(__builtin_mca_as_string) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result result = evaluate_expression(interpreter, arguments[0]);

    switch (result.value.type) {
        case M_T_INT: {
            int len = snprintf(NULL, 0, "%"PRId64, result.value.as.integer);

            // @Leak
            char *str = malloc(len + 1);

            snprintf(str, len + 1, "%"PRId64, result.value.as.integer);

            return (M_Value){ .type = M_T_STRING, .as.string.value = str, .as.string.value_length = len };
        }

        case M_T_FLOAT: {
            int len = snprintf(NULL, 0, "%f", result.value.as.floating);

            // @Leak
            char *str = malloc(len + 1);
            snprintf(str, len + 1, "%f", result.value.as.floating);

            return (M_Value){ .type = M_T_STRING, .as.string.value = str, .as.string.value_length = len };
        }

        case M_T_BOOL: {
            const char *bool_str = result.value.as.boolean ? "true" : "false";
            int len = strlen(bool_str);
            // @Leak
            char *str = malloc(len + 1);
            strcpy(str, bool_str);

            return (M_Value){ .type = M_T_STRING, .as.string.value = str, .as.string.value_length = len };
        }

        case M_T_STRING:
            return result.value;
        default:
            m_interpreter_error(arguments[0], "cannot cast '%s' to string", m_value_type_name(result.value.type));
    }

    return result.value;
}

BUILTIN(__builtin_mca_len) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result result = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_STRING | M_T_MAP | M_T_ARRAY);

    switch (result.value.type) {
        case M_T_STRING: return m_value_int(result.value.as.string.value_length);
        case M_T_MAP: return m_value_int(result.value.as.map->size);
        case M_T_ARRAY: return m_value_int(result.value.as.array->length);
        case M_T_MAP_IT:
        case M_T_INT:
        case M_T_FLOAT:
        case M_T_BOOL:
        case M_T_UNIT:
        case M_T_COUNT:
        case M_T_FN:
            assert(0 && "should never happen");
            break;
    }

    return m_value_unit(); // unreacheable
}

BUILTIN(__builtin_mca_as_srand) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result seed = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT);

    srand((unsigned int)seed.value.as.integer);

    return m_value_unit();
}

BUILTIN(__builtin_mca_as_rand) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result min_r = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT);
    M_Eval_Result max_r = m_result_expect_type(arguments[1], evaluate_expression(interpreter, arguments[1]), M_T_INT);

    int64_t min = min_r.value.as.integer;
    int64_t max = max_r.value.as.integer;

    if (min > max)
        m_interpreter_error(arguments[0], "invalid range for rand(). min (%ld) cannot be greater than max (%ld)", min, max);

    int64_t random = (rand() % (max - min + 1)) + min;

    return m_value_int(random);
}

BUILTIN(__builtin_mca_argc) {
    (void)caller;
    (void)arguments;
    (void)arguments_count;

    return m_value_int(interpreter->argc);
}

BUILTIN(__builtin_mca_argv) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result r = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_INT);

    int64_t index = r.value.as.integer;

    if (index < 0 || index >= interpreter->argc)
        m_interpreter_error(caller, "index %d is out of range. You have %d arguments.", index, interpreter->argc);

    int length = strlen(interpreter->argv[index]);

    return (M_Value){ .type = M_T_STRING, .allocated = true, .as.string.value = strndup(interpreter->argv[index], length), .as.string.value_length = length };
}

BUILTIN(__builtin_mca_select) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result data = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_STRING);
    M_Eval_Result from = m_result_expect_type(arguments[1], evaluate_expression(interpreter, arguments[1]), M_T_INT);
    M_Eval_Result to = m_result_expect_type(arguments[2], evaluate_expression(interpreter, arguments[2]), M_T_INT);

    if (from.value.as.integer < 0 || from.value.as.integer >= data.value.as.string.value_length)
        m_interpreter_error(arguments[1], "from '%d' is out of range. The size of the string is %d", from.value.as.integer, data.value.as.string.value_length);
    if (to.value.as.integer < 0 || to.value.as.integer >= data.value.as.string.value_length + 1)
        m_interpreter_error(arguments[2], "to '%d' is out of range. The size of the string is %d", to.value.as.integer, data.value.as.string.value_length);
    if (from.value.as.integer > to.value.as.integer)
        m_interpreter_error(arguments[1], "from '%d' cannot be greater than to '%d'", from.value.as.integer, to.value.as.integer);

    int64_t diff = to.value.as.integer - from.value.as.integer;

    // @Leak TODO: The chunk is leaked, we need to deallocate it in the garbage collector (when we eventually have one)
    char *chunk = strndup(data.value.as.string.value + from.value.as.integer, diff);

    return (M_Value){
        .type = M_T_STRING,
        .allocated = true,
        .as.string.value_length = diff,
        .as.string.value = chunk
    };
}

BUILTIN(__builtin_mca_ord) {
    (void)caller;
    (void)arguments_count;

    M_Eval_Result data = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_STRING);

    if (data.value.as.string.value_length != 1)
        m_interpreter_error(arguments[0], "ord() expects a string of length 1, got a string of length %d", data.value.as.string.value_length);

    return (M_Value){
        .type = M_T_INT,
        .allocated = false,
        .as.integer = (int64_t)data.value.as.string.value[0],
    };
}

BUILTIN(__builtin_mca_map_del) {
    (void)caller;
    (void)arguments_count;
    M_Eval_Result a0 = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_MAP);

    // for now, we will be able to have only integers and strings as keys
    M_Eval_Result a1 = m_result_expect_type(arguments[1], evaluate_expression(interpreter, arguments[1]), M_T_INT | M_T_STRING);

    void *key       = NULL;
    int key_type    = a1.value.type;
    size_t key_size = __get_map_key_data_and_size_helper(&a1.value, &key, true);

    bool deleted = mca_map_del(a0.value.as.map, key, key_size, key_type);

    return m_value_bool(deleted);
}

BUILTIN(__builtin_mca_map_clear) {
    (void)caller;
    (void)arguments_count;
    M_Eval_Result a0 = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_MAP);

    mca_map_clear(a0.value.as.map);

    return m_value_unit();
}

BUILTIN(__builtin_mca_append) {
    (void)caller;
    (void)arguments_count;

    M_Value arr = evaluate_expression(interpreter, arguments[0]).value;

    if (arr.type != M_T_ARRAY) {
        m_interpreter_error(arguments[0], "first argument to append must be an array");
        return m_value_unit();
    }

    M_Value val = evaluate_expression(interpreter, arguments[1]).value;
    M_Array *a = arr.as.array;

    if (a->length >= a->capacity) {
        a->capacity = a->capacity == 0 ? 8 : a->capacity * 2;
        a->items = realloc(a->items, sizeof(M_Value) * a->capacity);
    }

    a->items[a->length++] = val;

    return arr;
}

BUILTIN(__builtin_mca_format) {
    if (arguments_count <= 0) {
        m_interpreter_error(caller, "expected at least one argument but received %d", arguments_count);
    }

    size_t buffer_cap = 128;
    size_t buffer_len = 0;
    char *buffer = malloc(buffer_cap);

    if (!buffer) {
        m_interpreter_error(caller, "out of memory during format");
    }
    buffer[0] = '\0';

    for (int i = 0; i < arguments_count; i++) {
        M_Eval_Result result = m_result_expect_type(
            arguments[i],
            evaluate_expression(interpreter, arguments[i]),
            M_T_INT | M_T_STRING | M_T_FLOAT | M_T_BOOL
        );

        M_Value val = result.value;

        char temp_num_buf[128];
        const char *to_append = "";
        size_t append_len = 0;

        switch (val.type) {
            case M_T_INT:
                append_len = snprintf(temp_num_buf, sizeof(temp_num_buf), "%" PRId64, val.as.integer);
                to_append = temp_num_buf;
                break;
            case M_T_FLOAT:
                append_len = snprintf(temp_num_buf, sizeof(temp_num_buf), "%g", val.as.floating);
                to_append = temp_num_buf;
                break;
            case M_T_BOOL:
                to_append = val.as.boolean ? "true" : "false";
                append_len = strlen(to_append);
                break;
            case M_T_STRING:
                to_append = val.as.string.value;
                append_len = val.as.string.value_length;
                break;
            default:
                assert(0 && "__builtin_mca_format: should never happen");
                break;
        }

        if (buffer_len + append_len + 1 > buffer_cap) {
            while (buffer_len + append_len + 1 > buffer_cap) {
                buffer_cap *= 2;
            }
            buffer = realloc(buffer, buffer_cap);
            if (!buffer) {
                m_interpreter_error(caller, "out of memory during format reallocation");
            }
        }

        memcpy(buffer + buffer_len, to_append, append_len);
        buffer_len += append_len;
        buffer[buffer_len] = '\0';
    }

    M_Value final_result;
    final_result.allocated = true;
    final_result.type = M_T_STRING;

    final_result.as.string.value = buffer;
    final_result.as.string.value_length = buffer_len;

    return final_result;
}

BUILTIN(__builtin_mca_import) {
    (void)arguments_count;

    // TODO: later, cache the imported modules

    M_Eval_Result path_arg = m_result_expect_type(arguments[0], evaluate_expression(interpreter, arguments[0]), M_T_STRING);

    const char *raw_filepath = path_arg.value.as.string.value;

    if (path_arg.value.as.string.value_length == 0)
        m_interpreter_error(arguments[0], "invalid module path '%.*s'", path_arg.value.as.string.value_length, path_arg.value.as.string.value);

    char resolved_filepath[PATH_MAX];

    // if starts with a dot, it'll be considered as a relative local import
    if (strncmp(raw_filepath, ".", 1) == 0) {
        const char *caller_file = caller->location.filename;

        if (caller_file) {
            char *caller_file_copy = strdup(caller_file);
            char *base_dir = dirname(caller_file_copy);

            char combined_path[PATH_MAX];
            snprintf(combined_path, sizeof(combined_path), "%s/%s", base_dir, raw_filepath);

            if (realpath(combined_path, resolved_filepath) == NULL) {
                strncpy(resolved_filepath, combined_path, sizeof(resolved_filepath));
            }

            free(caller_file_copy);
        } else {
            strncpy(resolved_filepath, raw_filepath, sizeof(resolved_filepath));
        }
    } else {
        // may handle stdlib paths
        strncpy(resolved_filepath, raw_filepath, sizeof(resolved_filepath));
    }

    char *content = NULL;

    int size = read_entire_file_builtin(resolved_filepath, &content);

    // TODO: how are going to deal with error handling?
    switch (size) {
        case -1:
            m_interpreter_error(caller, "coult not open file '%s' due to: '%s'", resolved_filepath, strerror(errno));
            break;
        case -2:
            m_interpreter_error(caller, "could not allocate memory enough to read file %s due to: %s", resolved_filepath, strerror(errno));
            break;
        case -3:
            m_interpreter_error(caller, "could not read data from file '%s' due to: %s", resolved_filepath, strerror(errno));
            break;
        default:
            break;
    }

    M_Lexer lexer = m_lexer_create(resolved_filepath, content, size);

    if (lexer.errors > 0) m_interpreter_error(caller, "could not read module %s", resolved_filepath);

    M_Token *tokens = m_lexer_tokenize(&lexer);
    M_Ast *ast = parse_expression(resolved_filepath, tokens);

    if (ast->errors) m_interpreter_error(caller, "could not parse module %s", resolved_filepath);

    M_Interpreter *module_interpreter = m_interpreter_create(ast, 0, NULL);
    M_Value exported_module = m_interpreter_run(module_interpreter);

    // TODO: cleanup

    return exported_module;
}

// TODO: add a 'help' function that prints out the help for any builtin function
static M_Fn_Binding builtin_functions_bindings[] = {
    // TODO: check the amount of parameters inside the function

    // Math related
    BIND_FN("PI",    0, __builtin_mca_pi),  // TODO: should it become a constant variable (we don't have constant values yet)?
    BIND_FN("E",     0, __builtin_mca_e),   // TODO: should it become a constant variable (we don't have constant values yet)?
    BIND_FN("abs",   1, __builtin_mca_abs),
    BIND_FN("max",  -1, __builtin_mca_max),
    BIND_FN("min",  -1, __builtin_mca_min),
    BIND_FN("sin",   1, __builtin_mca_sin),
    BIND_FN("cos",   1, __builtin_mca_cos),
    BIND_FN("asin",  1, __builtin_mca_asin),
    BIND_FN("acos",  1, __builtin_mca_acos),
    BIND_FN("tan",   1, __builtin_mca_tan),
    BIND_FN("rad",   1, __builtin_mca_rad),
    BIND_FN("deg",   1, __builtin_mca_deg),
    BIND_FN("sqrt",  1, __builtin_mca_sqrt),
    BIND_FN("log",   1, __builtin_mca_log),
    BIND_FN("log10", 1, __builtin_mca_log10),
    BIND_FN("exp",   1, __builtin_mca_exp),
    BIND_FN("floor", 1, __builtin_mca_floor),
    BIND_FN("ceil",  1, __builtin_mca_ceil),
    BIND_FN("round", 1, __builtin_mca_round),

    // I/O / System related
    BIND_FN("println",          -1, __builtin_mca_println),
    BIND_FN("print",            -1, __builtin_mca_print),
    BIND_FN("read_entire_file",  1, __builtin_mca_read_entire_file),
    BIND_FN("exit",              1, __builtin_mca_exit),

    // language specifics
    BIND_FN("import",    1, __builtin_mca_import),

    BIND_FN("type",      1, __builtin_mca_type),
    BIND_FN("argc",      0, __builtin_mca_argc),
    BIND_FN("argv",      1, __builtin_mca_argv),
    BIND_FN("as_int",    1, __builtin_mca_as_int),
    BIND_FN("as_float",  1, __builtin_mca_as_float),
    BIND_FN("as_bool",   1, __builtin_mca_as_bool),
    BIND_FN("as_string", 1, __builtin_mca_as_string),
    BIND_FN("is_int",    1, __builtin_mca_is_int),
    BIND_FN("is_float",  1, __builtin_mca_is_float),
    BIND_FN("is_bool",   1, __builtin_mca_is_bool),
    BIND_FN("is_string", 1, __builtin_mca_is_string),
    BIND_FN("is_unit",   1, __builtin_mca_is_unit),
    BIND_FN("len",       1, __builtin_mca_len),

    // Strings
    BIND_FN("select",    3, __builtin_mca_select),
    BIND_FN("ord",       1, __builtin_mca_ord),
    BIND_FN("format",   -1, __builtin_mca_format), // format strings

    // Maps
    BIND_FN("map_del",   2, __builtin_mca_map_del),
    BIND_FN("map_clear", 1, __builtin_mca_map_clear),

    // Arrays
    BIND_FN("append",       2, __builtin_mca_append),

    // random
    BIND_FN("srand", 1, __builtin_mca_as_srand),
    BIND_FN("rand",  2, __builtin_mca_as_rand),

    // datetime related
    BIND_FN("time",        0, __builtin_mca_time),
    BIND_FN("year",        1, __builtin_mca_year),
    BIND_FN("month",       1, __builtin_mca_month),
    BIND_FN("date",        1, __builtin_mca_date),
    BIND_FN("day",         1, __builtin_mca_day),
    BIND_FN("hour",        1, __builtin_mca_hour),
    BIND_FN("minute",      1, __builtin_mca_minute),
    BIND_FN("second",      1, __builtin_mca_second),
    BIND_FN("millisecond", 0, __builtin_mca_millisecond),
};

static int builtin_functions_bindings_length = sizeof(builtin_functions_bindings) / sizeof(M_Fn_Binding);

static M_Fn_C_Impl resolve_builtin_function(M_Expression *expr) {
    // TODO: doing a linear search here doesn't seems to be a big problem (for a toy language)
    //       due to the fact we have just a few builtin functions.
    //       later on, we may improve this and use a proper hashmap pro max Iphone 15 Ultra pro Plus...
    for (int i = 0; i < builtin_functions_bindings_length; i++) {
        M_Fn_Binding signature = builtin_functions_bindings[i];

        if (signature.name_length != expr->Call.fn_name.value_length) continue;

        if (strncmp(signature.name, expr->Call.fn_name.value, signature.name_length) != 0) continue;

        // found a function that accepts N arguments
        if (signature.arguments_count == -1)
            return signature.c_impl;

        if (expr->Call.arguments_length > signature.arguments_count) {
            m_interpreter_error(expr, "too many arguments %s(...). expected %d but got %d", signature.name, signature.arguments_count, expr->Call.arguments_length);
        } else if (expr->Call.arguments_length < signature.arguments_count) {
            m_interpreter_error(expr, "too few arguments %s(...). expected %d but got %d", signature.name, signature.arguments_count, expr->Call.arguments_length);
        }

        // found a function that accepts this exact amount of arguments.
        // Note: data type will be checked later (lazy-checked-ish)
        return signature.c_impl;
    }

    return NULL;
}
