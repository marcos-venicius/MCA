//  Copyright (c) 2026 Marcos Venicius
// 
//  Author:           https://github.com/marcos-venicius
//  Original Repo:    https://github.com/marcos-venicius/clibs
#pragma once

#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>

typedef struct {
    uint8_t *buffer;
    size_t capacity;
    size_t offset;
    uint32_t alignment;
} Clibs_Arena;

Clibs_Arena* clibs_arena_create(size_t capacity, uint32_t alignment);
void* clibs_arena_alloc(Clibs_Arena *arena, size_t size);
void clibs_arena_reset(Clibs_Arena *arena);
void clibs_arena_destroy(Clibs_Arena *arena);

#define CLIBS_ARENA_DEFAULT_ALIGNMENT (sizeof(void*))
