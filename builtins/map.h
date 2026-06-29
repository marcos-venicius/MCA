#ifndef MCA_MAP_H_
#define MCA_MAP_H_

#include <stddef.h>

#define __MCA_MAP_CAP 1024 // 32^2 // should always be a power of 2

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
void              mca_map_free(M_Map *m);

M_Map_Iterator *mca_map_iterator(M_Map *m);
M_Map_Node     *mca_map_iterator_next(M_Map_Iterator *it);
bool            mca_map_iterator_finished(M_Map_Iterator *it);
void            mca_map_iterator_free(M_Map_Iterator *it);

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

static M_Map_Node *alloc_node(void *key, size_t key_size, int key_type, void *value, size_t value_size, int value_type) {
    M_Map_Node *node = (M_Map_Node*)malloc(sizeof(M_Map_Node));

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

static void free_node(M_Map_Node *node) {
    free(node->key.data);
    free(node->value.data);
    free(node);
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

void mca_map_set(M_Map *m, void *key, size_t key_size, int key_type, void *value, size_t value_size, int value_type) {
    uint32_t index = hash_key(key, key_size, key_type);

    if (m->nodes[index] == NULL) {
        m->nodes[index] = alloc_node(key, key_size, key_type, value, value_size, value_type);

        m->size++;
    } else {
        M_Map_Node *slow = NULL;
        M_Map_Node *fast = m->nodes[index];

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

bool mca_map_del(M_Map *m, void *key, size_t key_size, int key_type) {
    uint32_t index = hash_key(key, key_size, key_type);

    M_Map_Node *slow = NULL; 
    M_Map_Node *fast = m->nodes[index];

    while (fast) {
        if (cmp_keys(key, key_size, key_type, &fast->key)) {
            if (!slow) {
                m->nodes[index] = fast->next;
            } else {
                slow->next = fast->next;
            }

            free_node(fast);

            m->size--;

            return true;
        }

        slow = fast;
        fast = fast->next;
    }

    return false;
}

M_Map_Node_Entry *mca_map_find(M_Map *m, void *key, size_t key_size, int key_type) {
    uint32_t index = hash_key(key, key_size, key_type);

    M_Map_Node *head = m->nodes[index];

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
        M_Map_Node *head = m->nodes[i];

        while (head) {
            M_Map_Node *next = head->next;

            free_node(head);

            head = next;
        }
    }

    free(m);
}

M_Map_Iterator *mca_map_iterator(M_Map *m) {
    M_Map_Iterator *it = malloc(sizeof(M_Map_Iterator));

    it->map = m;
    it->idx = 0;
    it->it  = NULL;

    mca_map_iterator_next(it);

    return it;
}

M_Map_Node *mca_map_iterator_next(M_Map_Iterator *it) {
    if (mca_map_iterator_finished(it)) return NULL;

    if (it->it != NULL) {
        if (it->it->next != NULL) {
            return it->it = it->it->next;
        } else {
            it->idx++;
        }
    }

    while (it->idx < __MCA_MAP_CAP) {
        if (it->map->nodes[it->idx] != NULL) {
            it->it = it->map->nodes[it->idx];

            return it->it;
        }

        it->idx++;
    }

    it->it = NULL;

    return NULL;
}

inline bool mca_map_iterator_finished(M_Map_Iterator *it) {
    return it == NULL || (it->idx >= __MCA_MAP_CAP && it->it == NULL);
}

void mca_map_iterator_free(M_Map_Iterator *it) {
    if (it) free(it);
}

#endif // MCA_MAP_IMPLEMENTATION
