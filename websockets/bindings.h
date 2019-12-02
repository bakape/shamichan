#include <stddef.h>
#include <stdint.h>

// Wrapper for passing buffer references over the FFI
typedef struct {
    uint8_t* data;
    size_t size;
} WSBuffer;

// Register a websocket client with a unique ID and return any error as owned
// string.
//
// Error must be freed by caller, if not null.
char* ws_register_client(uint64_t id, WSBuffer ip);

// Remove client from registry
void ws_unregister_client(uint64_t id);

// Unref and potentially free a message on the Rust side
void ws_unref_message(WSBuffer* msg);

// Pass received message to Rust side. This operation never returns an error to
// simplify error propagation. All errors are propagated back to Go only using
// ws_close_client.
void ws_receive_message(uint64_t client_id, WSBuffer msg);
