#include "connection.hh"
#include "../../brunhild/mutations.hh"
#include "../json.hh"
#include "../lang.hh"
#include "../state.hh"
#include "../util.hh"
#include "sync.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <string_view>

using nlohmann::json;
using std::string;

FSM<ConnState, ConnEvent>* conn_SM = nullptr;

static void on_open()
{
    log_exceptions([]() { conn_SM->feed(ConnEvent::open); });
}

static void on_close()
{
    log_exceptions([]() { conn_SM->feed(ConnEvent::close); });
}

static void on_error(string msg)
{
    console::log(msg);
    log_exceptions([]() { conn_SM->feed(ConnEvent::close); });
}

// Prepend type information to stringified message
static string encode_message(Message type, const string& msg)
{
    string s;
    s.reserve(msg.size() + 2);
    const uint8_t i = static_cast<uint8_t>(type);
    if (i < 10) {
        s += '0';
    }
    s += std::to_string(i) + msg;
    return s;
}

// Handler for messages received from the server.
// extracted specifies, the mesage was extracted from a larger concatenated
// message.
static void on_message(string data_str, bool extracted)
{
    log_exceptions([=]() {
        auto data = std::string_view(data_str);
        if (debug) {
            string s;
            s.reserve(data.size() + 3);
            if (extracted) {
                s += '\t';
            }
            s += "< " + string(data);
            console::log(s);
        }

        const Message type
            = static_cast<Message>(std::stoul(string(data.substr(0, 2))));
        data = data.substr(2);
        switch (type) {
        case Message::synchronise:
            load_posts(data);
            conn_SM->feed(ConnEvent::sync);
            break;
        case Message::concat: {
            // Split several concatenated messages
            size_t last = 0;
            while (1) {
                const size_t i = data.find(char(0), last);
                on_message(string(data.substr(last, i)), true);
                if (i == -1) {
                    break;
                }
                last = i + 1;
            }
            return;
        }
        default:
            console::warn("unknown websocket message: "
                + encode_message(type, string(data)));
            return;
        }
    });
}

static void retry_to_connect()
{
    log_exceptions([]() { conn_SM->feed(ConnEvent::retry); });
}

// Work around browser slowing down/suspending tabs and keep the FSM up to date
// with the actual status.
static void on_window_focus()
{
    if (!emscripten::val::global("navigator")["online"].as<bool>()) {
        return;
    }
    switch (conn_SM->state()) {
    // Ensure still connected, in case the computer went to sleep or
    // hibernate or the mobile browser tab was suspended.
    case ConnState::synced:
        send_message(Message::NOP, "");
    case ConnState::desynced:
        break;
    default:
        conn_SM->feed(ConnEvent::retry);
    }
}

EMSCRIPTEN_BINDINGS(module_conn)
{
    using namespace emscripten;

    function("on_socket_open", &on_open);
    function("on_socket_close", &on_close);
    function("on_socket_message", &on_message);
    function("retry_to_connect", &retry_to_connect);
}

static void connect()
{
    EM_ASM({
        var path = (location.protocol == 'https:' ? 'wss' : 'ws') + '://'
            + location.host + '/api/socket';
        var s = window.__socket = new WebSocket(path);
        s.onopen = function() { Module.on_socket_open(); };
        s.onmessage = function(e) { Module.on_socket_message(e.data, false); };
        s.onclose = function() { Module.on_socket_close(); };
        s.onerror = function(e)
        {
            console.error(e);
            Module.on_socket_close();
        };
    });
}

void send_message(Message type, string msg)
{
    const string s = encode_message(type, msg);
    if (debug) {
        console::log("< " + s);
    }
    EM_ASM_INT({ window.__socket.send(UTF8ToString($0)); }, s.c_str());
}

// Render connection status indicator
static void render_status(SyncStatus status)
{
    string s;
    if (status != SyncStatus::hide) {
        s = lang->sync[static_cast<size_t>(status)];
    }
    brunhild::set_inner_html("sync", s);
}

// Set synced IP count to n
static void render_sync_count(unsigned int n)
{
    string s;
    if (n) {
        s = std::to_string(n);
    }
    brunhild::set_inner_html("sync_counter", s);
}

// Prepare for synchronization with server
static ConnState prepare_to_sync()
{
    render_status(SyncStatus::connecting);
    send_sync_request();
    EM_ASM({
        window.__connection_attempt_timer
            = setTimeout(window.reset_connection_attempts, 10000);
    });
    return ConnState::syncing;
}

// Reset timer and count for connection attempts
static void reset_connection_attempts()
{
    EM_ASM({ window.reset_connection_attempts(); });
}

void init_connectivity()
{
    conn_SM = new FSM<ConnState, ConnEvent>(ConnState::loading);

    // Define some JS-side functions and listeners
    EM_ASM({
        window.reset_connection_attempts = function()
        {
            if (window.__connection_attempt_timer) {
                clearTimeout(window.__connection_attempt_timer);
                window.__connection_attempt_timer = 0;
            }
            window.__connection_attempt_count = 0;
        };

        window.addEventListener('online', function() {
            window.reset_connection_attempts();
            Module.retry_to_connect();
        });
        window.addEventListener('offline', function() {
            if (window.__socket) {
                window.__socket.close();
                window.__socket = null;
            }
        });
    });

    // Define transition rules for the connection FSM
    conn_SM->act(ConnState::loading, ConnEvent::start, []() {
        render_status(SyncStatus::connecting);
        connect();
        return ConnState::connecting;
    });
    conn_SM->act(ConnState::connecting, ConnEvent::open, &prepare_to_sync);
    conn_SM->act(ConnState::reconnecting, ConnEvent::open, &prepare_to_sync);
    conn_SM->act(ConnState::syncing, ConnEvent::sync, []() {
        render_status(SyncStatus::synced);
        return ConnState::synced;
    });
    conn_SM->wild_act(ConnEvent::close, []() {
        render_status(SyncStatus::disconnected);
        EM_ASM({
            window.reset_connection_attempts();

            // Wait maxes out at ~1min
            var wait = Math.min(
                Math.floor(++window.__connection_attempt_count / 2), 12);
            wait = 500 * Math.pow(1.5, wait);
            window.__connection_attempt_timer
                = setTimeout(Module.retry_to_connect, wait);
        });
        return ConnState::dropped;
    });
    conn_SM->wild_act(ConnEvent::error, []() {
        reset_connection_attempts();
        render_status(SyncStatus::desynced);
        return ConnState::desynced;
    });
    conn_SM->act(ConnState::dropped, ConnEvent::retry, []() {
        if (!emscripten::val::global("navigator")["online"].as<bool>()) {
            return ConnState::dropped;
        }
        connect();
        render_status(SyncStatus::connecting);
        return ConnState::reconnecting;
    });
}
