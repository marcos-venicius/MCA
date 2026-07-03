#pragma once

#include "./ht.h"

typedef struct M_Environment M_Environment;

struct M_Environment {
    ht_t *variables;

    M_Environment *parent;
};
