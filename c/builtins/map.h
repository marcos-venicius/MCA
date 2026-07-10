#pragma once

#include <stddef.h>
#include <stdbool.h>

#define __MCA_MAP_CAP 4096 // 32^2 // should always be a power of 2

typedef struct {
    void  *data; 
    size_t size;
    int    type;
} M_Map_Node_Entry;

typedef struct M_Map_Node M_Map_Node;

struct M_Map_Node {
    M_Map_Node_Entry key;
    M_Map_Node_Entry value;

    M_Map_Node *next;
};

typedef struct {
    M_Map_Node *nodes[__MCA_MAP_CAP];
    int         size;
} M_Map;

typedef struct {
    M_Map      *map;
    M_Map_Node *it;
    int         idx;
} M_Map_Iterator;

M_Map            *mca_map_init();
void              mca_map_set(M_Map *m, void *key, size_t key_size, int key_type, void *value, size_t value_size, int value_type);
bool              mca_map_del(M_Map *m, void *key, size_t key_size, int key_type);
M_Map_Node_Entry *mca_map_find(M_Map *m, void *key, size_t key_size, int key_type);
void              mca_map_clear(M_Map *m);
void              mca_map_free(M_Map *m);

M_Map_Iterator *mca_map_iterator(M_Map *m);
M_Map_Node     *mca_map_iterator_next(M_Map_Iterator *it);
bool            mca_map_iterator_finished(M_Map_Iterator *it);
void            mca_map_iterator_free(M_Map_Iterator *it);
