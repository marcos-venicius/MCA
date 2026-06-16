//  Copyright (c) 2026 Marcos Venicius
// 
//  Author:           https://github.com/marcos-venicius
//  Original Repo:    https://github.com/marcos-venicius/clibs

#ifndef CLIBS_ARENA
#define CLIBS_ARENA

#include <stddef.h>
#include <stdint.h>

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

#endif // CLIBS_ARENA

#ifdef CLIBS_ARENA_IMPLEMENTATION

#include <stdlib.h>
#include <stdio.h>

Clibs_Arena *clibs_arena_create(size_t capacity, uint32_t alignment) {
    Clibs_Arena *arena = (Clibs_Arena *)malloc(sizeof(Clibs_Arena));

    if (!arena) return NULL;

    arena->buffer = (uint8_t *)malloc(capacity);

    if (!arena->buffer) {
        free(arena);
        return NULL;
    }

    arena->alignment = alignment;
    arena->capacity = capacity;
    arena->offset = 0;

    return arena;
}

void *clibs_arena_alloc(Clibs_Arena *arena, size_t size) {
    size_t padding = 0;
    uintptr_t current_ptr = (uintptr_t)(arena->buffer + arena->offset);
    uintptr_t remainder = current_ptr % arena->alignment;

    if (remainder != 0) {
        padding = arena->alignment - remainder;
    }

    if (arena->offset + padding + size > arena->capacity) {
        fprintf(stderr, "[CLIBS_ARENA_ERROR]: Arena ran out of memory\n");
        exit(1);
    }

    void *allocated_ptr = arena->buffer + arena->offset + padding;

    arena->offset += padding + size;

    return allocated_ptr;
}

void clibs_arena_reset(Clibs_Arena *arena) {
    arena->offset = 0;
}

void clibs_arena_destroy(Clibs_Arena *arena) {
    if (arena) {
        free(arena->buffer);
        free(arena);
    }
}

#endif // CLIBS_ARENA_IMPLEMENTATION
