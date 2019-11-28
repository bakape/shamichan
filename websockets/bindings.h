#include <stddef.h>
#include <stdint.h>

// Websocket message passed to Go
typedef struct {
    uint8_t* data;
    size_t size;
} WSMessage;

// Register a websocket client with a unique ID and return any error as owned
// string.
//
// Error must be freed by caller, if not null.
char* ws_register_client(uint64_t id, const char* ip);

// Remove client from registry
void ws_unregister_client(uint64_t id);

// Unref and potentially free a message on the Rust side
void ws_unref_message(const WSMessage* msg);
