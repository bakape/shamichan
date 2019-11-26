#include <stddef.h>
#include <stdint.h>

// Websocket message passed to Go
typedef struct {
    uint8_t* data;
    size_t size;

    // Used to unref and potentially free the message on the Rust side
    void* handle;
} ws_message;

// Register a websocket client by ID and its IP as a string reference.
//
// Returns any error encoded as a C string.
char* ws_register_client(uint64_t id, const char* ip);

// Remove client from registry
void ws_unregister_client(uint64_t id);

// Used to unref and potentially free a message on the Rust side
void ws_unref_message(const void* handle);

// Write message to registered client
extern void ws_write_message(uint64_t client_id, const ws_message msg);

// Forcefully close a client with an optional owned error message
extern void ws_close_client(uint64_t client_id, char* err);
