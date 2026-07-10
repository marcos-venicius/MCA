#include "./map.h"

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

static uint32_t hash_key(void *data, size_t size, int type) {
    const unsigned char *bytes = (const unsigned char*)data;
    uint32_t hash = 2166136261u;

    // Mix in type and size to ensure different types/sizes with same prefix don't collide
    hash ^= (uint32_t)type;
    hash *= 16777619u;
    
    hash ^= (uint32_t)size;
    hash *= 16777619u;

    for (size_t i = 0; i < size; i++) {
        hash ^= bytes[i];
        hash *= 16777619u;
    }

    return hash & (__MCA_MAP_CAP - 1);
}

static M_Map_Node *alloc_node(void *key, size_t key_size, int key_type, void *value, size_t value_size, int value_type) {
    // Allocate node and key data in a single contiguous block to save a malloc()
    M_Map_Node *node = (M_Map_Node*)malloc(sizeof(M_Map_Node) + key_size);

    node->key.data = (char*)node + sizeof(M_Map_Node);
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
    // node->key.data is part of the node block, so only free value.data and node
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

void mca_map_clear(M_Map *m) {
    for (int i = 0; i < __MCA_MAP_CAP; i++) {
        M_Map_Node *head = m->nodes[i];

        while (head) {
            M_Map_Node *next = head->next;
            free_node(head);
            head = next;
        }

        m->nodes[i] = NULL;
    }

    m->size = 0;
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
    M_Map_Iterator *it = ( M_Map_Iterator *)malloc(sizeof(M_Map_Iterator));

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