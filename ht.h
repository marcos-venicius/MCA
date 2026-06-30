#pragma once

#define __cearch_ht_capacity 1019

typedef struct ht_node_t ht_node_t;

struct ht_node_t {
    struct {
        char *key;
        int length;
    } name;
    void *data;
    ht_node_t *next;
};

typedef struct {
    int length;
    int data_size;
    ht_node_t *nodes[__cearch_ht_capacity];
} ht_t;

typedef struct {
    ht_t      *ht;  // hash table
    ht_node_t *it;  // current node on iteration
    int        idx; // current nodes array position on iteration
} ht_iterator_t;

ht_t *ht_init(int data_size);
void ht_add(ht_t *ht, const char *key, void *data);
void *ht_find(ht_t *ht, const char *key);
void ht_free(ht_t *ht);

// ITERATOR STUFF
ht_iterator_t ht_iterator(ht_t *ht);
ht_node_t *ht_iterator_next(ht_iterator_t *it);