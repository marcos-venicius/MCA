#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include <stdlib.h>
#include <errno.h>
#include "./lexer.h"
#include "./log.h"

typedef struct {
    char *input_file_name;
    char *output_file_name;
    char *input_as_string;
} ProgramArguments;

void usage(FILE *stream, const char *program_name) {
    fprintf(stream, "USAGE: %s -i <file> -o output.asm \n\n", program_name);
    fprintf(stream, "-i                  (required or -s) input file name\n");
    fprintf(stream, "-o                  (optional, by default is a.asm) output file name\n");
    fprintf(stream, "-s                  (required or -i) input as string\n");
    fprintf(stream, "-h                  show this help\n");
    fprintf(stream, "\n");
}

bool cmp_arg(const char *a, const char *b) {
    size_t sa = strlen(a);
    size_t sb = strlen(b);

    if (sa != sb) return false;

    for (size_t i = 0; i < sa; ++i)
        if (a[i] != b[i]) return false;

    return true;
}

char *get_argument_value(const char *program_name, const char *arg, int *argc, char ***argv) {
    (*argc)--;

    if (*argc <= 0) {
        usage(stderr, program_name);
        fprintf(stderr, "ERROR: missing value for argument %s\n", arg);
        exit(1);
    }

    char *value = *(++(*argv));

    if (*value == '-') {
        usage(stderr, program_name);
        fprintf(stderr, "ERROR: missing value for argument %s\n", arg);
        exit(1);
    }

    return value;
}

int read_file_content(const char *filename, char **output) {
    FILE *fptr = fopen(filename, "r");

    if (fptr == NULL) {
        fprintf(stderr, "ERROR: could not open file %s due to: %s\n", filename, strerror(errno));
        return -1;
    }

    fseek(fptr, 0, SEEK_END);
    size_t file_size = ftell(fptr);
    rewind(fptr);

    *output = malloc(file_size * (sizeof(char) + 1));

    if (*output == NULL) {
        fclose(fptr);

        fprintf(stderr, "ERROR: could not allocate memory enough to read the file %s due to: %s\n", filename, strerror(errno));

        return -1;
    }

    size_t read_bytes = fread(*output, sizeof(char), file_size, fptr);

    if (read_bytes != file_size) {
        free(*output);
        fclose(fptr);

        fprintf(stderr, "ERROR: coult not read data from file: %s\n", strerror(errno));

        return -1;
    }

    fclose(fptr);

    (*output)[file_size] = '\0';

    return file_size;
}

void compile_math(const char *filename, const char *string, const size_t string_size) {
    LOG("[*] compiling math\n");

    M_Lexer lexer = m_lexer_create(filename, string, string_size);

    M_Token *tokens = m_lexer_tokenize(&lexer);

    if (tokens == NULL) {
        LOG("[*] There is no tokens\n");
        return;
    }

    if (is_log_enabled()) {
        printf("TOKENS: \n");
        for (M_Token *token = tokens; token != NULL; token = token->next) {
            printf("    <Token value=[%.*s] kind=[%s] />\n", (int)token->size, token->value, m_lexer_token_kind_display_name(token->kind));
        }
    }
}

int main(int argc, char **argv) {
    init_logging();

    argc--;
    const char *program_name = *(argv++);

    ProgramArguments p_arguments = {0};

    p_arguments.output_file_name = "a.asm";

    while (argc > 0) {
        const char *arg = *argv;

        if (*arg != '-') {
            usage(stderr, program_name);
            fprintf(stderr, "ERROR: invalid argument: %s\n", arg);
            return 1;
        }

        if (cmp_arg(arg, "-h")) {
            usage(stdout, program_name);
            return 0;
        } else if (cmp_arg(arg, "-i")) {
            p_arguments.input_file_name = get_argument_value(program_name, arg, &argc, &argv);
        } else if (cmp_arg(arg, "-o")) {
            p_arguments.output_file_name = get_argument_value(program_name, arg, &argc, &argv);
        } else if (cmp_arg(arg, "-s")) {
            p_arguments.input_as_string = get_argument_value(program_name, arg, &argc, &argv);
        } else {
            usage(stderr, program_name);
            fprintf(stderr, "ERROR: unrecognized argument: %s\n", arg);
            return 1;
        }

        argv++;
        argc--;
    }

    if (p_arguments.input_file_name == NULL && p_arguments.input_as_string == NULL) {
        usage(stderr, program_name);
        fprintf(stderr, "ERROR: please, provide either -s or -i as an input\n");
        return 1;
    }


    if (p_arguments.input_file_name != NULL) {
        char *input;
        int size;

        if ((size = read_file_content(p_arguments.input_file_name, &input)) < 0) return 1;

        compile_math(p_arguments.input_file_name, input, size);
        free(input);
    } else {
        compile_math(NULL, p_arguments.input_as_string, strlen(p_arguments.input_as_string));
    }

    return 0;
}
