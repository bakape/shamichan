#pragma once

#include "../fsm.hh"

// Websocket connection and synchronization with server states
enum class SyncStatus {
    disconnected,
    connecting,
    syncing,
    synced,
    desynced,
    hide
};

// States of the connection finite state machine
enum class ConnState {
    loading,
    connecting,
    syncing,
    synced,
    dropped,
    desynced
};

// Events passable to the connection FSM
enum class ConnEvent { start, open, close, retry, error, sync, switch_sync };

// Finite state machine for managing websocket connectivity
extern FSM<ConnState, ConnEvent> conn_SM;

// Message types of the WebSocket communication protocol
enum class Message : uint8_t {
    invalid,

    // 1 - 29 modify post model state
    _,
    insert_post,
    append,
    backspace,
    splice,
    close_post,
    __,
    insert_image,
    spoiler,
    delete_post,
    banned,
    delete_image,

    // >= 30 are miscellaneous and do not write to post models
    synchronise = 30,
    reclaim,

    // Send new post ID to client
    post_ID,

    // Concatenation of multiple websocket messages to reduce transport overhead
    concat,

    // Invokes no operation on the server. Used to test the client's connection
    // in situations, when you can't be certain the client is still connected.
    NOP,

    // Transmit current synced IP count to client
    sync_count,

    // Send current server Unix time to client
    server_time,

    // Redirect the client to a specific board
    redirect,

    // Send a notification to a client
    notification,

    // Notification about needing a captcha on the next post allocation
    captcha,

    // Data concerning random video feed
    megu_tv,
};

// Initialize websocket connectivity module
void init_connectivity();

// Send a websocket message the server
void send_message(Message, std::string);
