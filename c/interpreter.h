#pragma once

#include <stdio.h>

#include "./ast.h"
#include "./ht.h"
#include "./builtins/map.h"
#include "./env.h"


typedef struct {
    M_Environment *global_environment;

    M_Environment *current_environment;

    M_Ast *program;

    int          argc;
    const char **argv;

    FILE *io_out; // default is 'C stdout'
    FILE *io_err; // default is 'C stderr'
    FILE *io_in;  // default is 'C stdin'
} M_Interpreter;

typedef enum {
    M_T_UNIT   = 1 << 0,
    M_T_INT    = 1 << 1,
    M_T_FLOAT  = 1 << 2,
    M_T_BOOL   = 1 << 3,
    M_T_STRING = 1 << 4,
    M_T_MAP    = 1 << 5,
    M_T_MAP_IT = 1 << 6,
    M_T_FN     = 1 << 7,
    M_T_ARRAY  = 1 << 8,
    M_T_COUNT
} M_Value_Type;

typedef struct M_Array M_Array;

struct M_Array {
    struct M_Value *items;
    int length;
    int capacity;
};

typedef union {
    double           floating;
    int64_t          integer;
    bool             boolean;
    M_String         string;
    M_Map           *map;
    M_Map_Iterator  *map_it;
    M_Expression    *fn;
    M_Array         *array;
} M_Value_Union;

typedef struct M_Value M_Value;

struct M_Value {
    // if the value is just a view or actually allocated
    bool allocated;

    M_Value_Type type;

    M_Value_Union as;
};

typedef enum {
    M_CTRL_NORMAL,
    M_CTRL_BREAK,
    M_CTRL_RETURN,
} M_Control_Flow;

typedef struct {
    M_Value value;
    M_Control_Flow flow;
} M_Eval_Result;

M_Interpreter *m_interpreter_create(M_Ast *program, int argc, const char **argv);
void m_interpreter_set_stdin(M_Interpreter *interpreter, FILE *stream);
void m_interpreter_set_stdout(M_Interpreter *interpreter, FILE *stream);
void m_interpreter_set_stderr(M_Interpreter *interpreter, FILE *stream);
M_Value m_interpreter_run(M_Interpreter *interpreter);
void m_interpreter_free(M_Interpreter *interpreter);