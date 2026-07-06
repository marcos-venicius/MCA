#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <string.h>

#include "./io.h"

int read_entire_file_builtin(const char *filename, char **output) {
    FILE *fptr = fopen(filename, "r");

    if (fptr == NULL) {
        return -1;
    }

    fseek(fptr, 0, SEEK_END);
    size_t file_size = ftell(fptr);
    rewind(fptr);

    *output = malloc(file_size * (sizeof(char) + 1));

    if (*output == NULL) {
        fclose(fptr);

        fprintf(stderr, "ERROR: could not allocate memory enough to read the file %s due to: %s\n", filename, strerror(errno));

        return -2;
    }

    size_t read_bytes = fread(*output, sizeof(char), file_size, fptr);

    if (read_bytes != file_size) {
        free(*output);
        fclose(fptr);

        fprintf(stderr, "ERROR: coult not read data from file: %s\n", strerror(errno));

        return -3;
    }

    fclose(fptr);

    (*output)[file_size] = '\0';

    return file_size;
}