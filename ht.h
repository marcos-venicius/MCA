#ifndef CLIBS_HT
#define CLIBS_HT

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

#ifdef CLIBS_HT_IMPLEMENTATION

#include <stdlib.h>
#include <string.h>
#include <assert.h>

static int ht_hash_key(const char *key) {
    const unsigned char *str = (const unsigned char*)key;

    unsigned long hash = 5381;
    int c;

    while ((c = *str++)) hash = ((hash << 5) + hash) + c;

    return hash % __cearch_ht_capacity;
}

static ht_node_t *ht_alloc_node(ht_t *ht, const char *key, int key_length, void *data) {
    ht_node_t *node = (ht_node_t *)malloc(sizeof(ht_node_t));

    // I'm using strdup (which includes the null-byte) because
    // we may want provide a loop for the client with all keys and values.
    // I don't want to mess up and let a segfault happen just because
    // the called is not concerned about this.
    node->name.key = strdup(key);
    node->name.length = key_length;
    node->next = NULL;
    node->data = malloc(ht->data_size);

    memcpy(node->data, data, ht->data_size);

    return node;
}

static inline void ht_free_node(ht_node_t *node) {
    free(node->data);
    free(node->name.key);
    free(node);
}

ht_t *ht_init(int data_size) {
    ht_t *ht = (ht_t *)calloc(1, sizeof(ht_t));

    assert(data_size > 0 && "'data_size' should be greater than zero");

    ht->data_size = data_size;

    return ht;
}

void ht_add(ht_t *ht, const char *key, void *data) {
    // TODO: we're not rehashing the keys,
    //       we may do it in the future, but for the current
    //       purpose of this DS we don't need it.
    int index = ht_hash_key(key);
    int key_length = strlen(key);

    if (ht->nodes[index] == NULL) {
        ht->nodes[index] = ht_alloc_node(ht, key, key_length, data);
    } else {
        ht_node_t *slow = NULL;
        ht_node_t *fast = ht->nodes[index];

        while (fast != NULL) {
            // exact same key
            if (fast->name.length == key_length && (memcmp(fast->name.key, key, key_length) == 0)) {
                // substituted node with exact same key
                memcpy(fast->data, data, ht->data_size);
                return;
            }

            slow = fast;
            fast = fast->next;
        }

        // added new node at the end
        slow->next = ht_alloc_node(ht, key, key_length, data);
    }
}

void *ht_find(ht_t *ht, const char *key) {
    int index = ht_hash_key(key);
    int key_length = strlen(key);

    ht_node_t *curr = ht->nodes[index];

    while (curr != NULL) {
        if (curr->name.length == key_length && (memcmp(curr->name.key, key, key_length) == 0))
            return curr->data;

        curr = curr->next;
    }

    return NULL;
}

void ht_free(ht_t *ht) {
    // TODO: use arena?
    for (int i = 0; i < __cearch_ht_capacity; i++) {
        ht_node_t *head = ht->nodes[i];

        while (head != NULL) {
            ht_node_t *next = head->next;

            ht_free_node(head);

            head = next;
        }
    }

    free(ht);
}

// ITERATOR STUFF

ht_iterator_t ht_iterator(ht_t *ht) {
    return (ht_iterator_t){
        .ht = ht,
        .it = NULL,
        .idx = 0
    };
}

ht_node_t *ht_iterator_next(ht_iterator_t *it) {
    if (it == NULL || (it->idx >= __cearch_ht_capacity && it->it == NULL)) return NULL;

    if (it->it != NULL) {
        if (it->it->next != NULL)
            return it->it = it->it->next;
        else
            it->idx++;
    }

    while (it->idx < __cearch_ht_capacity) {
        if (it->ht->nodes[it->idx] != NULL) {
            it->it = it->ht->nodes[it->idx];

            return it->it;
        }

        it->idx++;
    }

    it->it = NULL;

    return NULL;
}

#endif // CLIBS_HT_IMPLEMENTATION

#endif // CLIBS_HT
