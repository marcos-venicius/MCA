#include "./log.h"
#include <stdlib.h>

static bool logging_is_enabled = false;

void init_logging() {
    const char *env = getenv("MCA_LOG_ENABLED");

    if (env != NULL && *env == '1') {
        logging_is_enabled = true;
    }
}

bool is_log_enabled() {
    return logging_is_enabled;
}
