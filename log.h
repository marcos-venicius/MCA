#ifndef LOG_H_
#define LOG_H_
#include <stdio.h>
#include <stdbool.h>

#define LOG(x, ...) do {\
    if (is_log_enabled()) fprintf(stderr, x, ##__VA_ARGS__); \
} while (0)

void init_logging();
bool is_log_enabled();

#endif // LOG_H_
