#ifndef MCA_MAP_H_
#define MCA_MAP_H_

#include <stddef.h>

#define __MCA_MAP_CAP 1024 // 32^2 // should always be a power of 2

typedef struct {
    void  *data; 
    size_t size;
    int    type;
} M_Map_Node_Entry;

typedef struct mca_map_node_t mca_map_node_t;

struct mca_map_node_t {
    M_Map_Node_Entry key;
    M_Map_Node_Entry value;

    mca_map_node_t *next;
};

typedef struct {
    mca_map_node_t *nodes[__MCA_MAP_CAP];
    int             size;
} M_Map;

M_Map            *mca_map_init();
void              mca_map_add(M_Map *m, void *key, size_t key_size, int key_type, void *value, size_t value_size, int value_type);
M_Map_Node_Entry *mca_map_find(M_Map *m, void *key, size_t key_size, int key_type);
void              mca_map_free(M_Map *m);

#endif // MCA_MAP_H_

#ifdef MCA_MAP_IMPLEMENTATION

#include <stdlib.h> // malloc, free
#include <stdint.h> // uint32_t
#include <string.h> // memcpy

static uint32_t hash_key(void *data, size_t size, int type) {
    const unsigned char *bytes = (const unsigned char*)data;

    unsigned long hash = 5381;

    hash = ((hash << 5) + hash) + (uint32_t)size;
    hash = ((hash << 5) + hash) + (uint32_t)type;

    for (size_t i = 0; i < size; i++)
        hash = ((hash << 5) + hash) + bytes[i];

    return hash & (__MCA_MAP_CAP - 1);
}

static mca_map_node_t *alloc_node(void *key, size_t key_size, int key_type, void *value, size_t value_size, int value_type) {
    mca_map_node_t *node = (mca_map_node_t*)malloc(sizeof(mca_map_node_t));

    node->key.data = malloc(key_size);
    node->key.size = key_size;
    node->key.type = key_type;

    node->value.data = malloc(value_size);
    node->value.size = value_size;
    node->value.type = value_type;

    memcpy(node->key.data, key, key_size);
    memcpy(node->value.data, value, value_size);

    node->next = NULL;

    return node;
}

static int cmp_keys(void *key, size_t key_size, int key_type, M_Map_Node_Entry *entry) {
    if (entry->size != key_size) return 0;
    if (entry->type != key_type) return 0;

    return memcmp(key, entry->data, key_size) == 0;
}

M_Map *mca_map_init() {
    M_Map *map = (M_Map*)calloc(1, sizeof(M_Map));

    return map;
}

void mca_map_add(M_Map *m, void *key, size_t key_size, int key_type, void *value, size_t value_size, int value_type) {
    uint32_t index = hash_key(key, key_size, key_type);

    if (m->nodes[index] == NULL) {
        m->nodes[index] = alloc_node(key, key_size, key_type, value, value_size, value_type);

        m->size++;
    } else {
        mca_map_node_t *slow = NULL;
        mca_map_node_t *fast = m->nodes[index];

        while (fast) {
            if (cmp_keys(key, key_size, key_type, &fast->key)) {
                // Only reallocate if the new value needs more space
                if (value_size > fast->value.size) {
                    free(fast->value.data);
                    fast->value.data = malloc(value_size);
                }

                fast->value.size = value_size;
                fast->value.type = value_type;
                memcpy(fast->value.data, value, value_size);
                return;
            }

            slow = fast;
            fast = fast->next;
        }

        slow->next = alloc_node(key, key_size, key_type, value, value_size, value_type);
        m->size++;
    }
}

M_Map_Node_Entry *mca_map_find(M_Map *m, void *key, size_t key_size, int key_type) {
    uint32_t index = hash_key(key, key_size, key_type);

    mca_map_node_t *head = m->nodes[index];

    while (head) {
        if (cmp_keys(key, key_size, key_type, &head->key))
            return &head->value;

        head = head->next;
    }

    return NULL;
}

void mca_map_free(M_Map *m) {
    // TODO: use arena?

    for (int i = 0; i < __MCA_MAP_CAP; i++) {
        mca_map_node_t *head = m->nodes[i];

        while (head) {
            mca_map_node_t *next = head->next;

            free(head->key.data);
            free(head->value.data);
            free(head);

            head = next;
        }
    }

    free(m);
}

#endif // MCA_MAP_IMPLEMENTATION
