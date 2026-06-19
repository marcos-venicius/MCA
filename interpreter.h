#ifndef EVALUATOR_H_
#define EVALUATOR_H_

#include <stdio.h>

#include "./ast.h"
#include "./ht.h"

typedef struct M_Interpreter_Environment M_Interpreter_Environment;

struct M_Interpreter_Environment {
    ht_t *variables;

    M_Interpreter_Environment *parent;
};

typedef struct {
    M_Interpreter_Environment *global_environment;

    M_Interpreter_Environment *current_environment;

    M_Ast *program;

    FILE *io_out; // default is 'C stdout'
    FILE *io_err; // default is 'C stderr'
    FILE *io_in;  // default is 'C stdin'
} M_Interpreter;

typedef enum {
    M_CTRL_NORMAL,
    M_CTRL_BREAK,
} M_Control_Flow;

typedef struct {
    double value;
    M_Control_Flow flow;
} M_Eval_Result;

M_Interpreter *m_interpreter_create(M_Ast *program);
void m_interpreter_set_stdin(M_Interpreter *interpreter, FILE *stream);
void m_interpreter_set_stdout(M_Interpreter *interpreter, FILE *stream);
void m_interpreter_set_stderr(M_Interpreter *interpreter, FILE *stream);
double m_interpreter_run(M_Interpreter *interpreter);
void m_interpreter_free(M_Interpreter *interpreter);

// TODO: just for testing purposes. I should think of a better
// way to do this
// @deprecated
M_Eval_Result evaluate_expression(M_Expression *expression);

#endif // EVALUATOR_H_
