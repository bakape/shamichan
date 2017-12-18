#include "connection.hh"
#include "../../brunhild/mutations.hh"
#include "../json.hh"
#include "../lang.hh"
#include "../state.hh"
#include "../util.hh"
#include "sync.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <sstream>

using nlohmann::json;
using std::string;

FSM<ConnState, ConnEvent>* conn_SM = nullptr;

static void on_open() { conn_SM->feed(ConnEvent::error); }

static void on_close() { conn_SM->feed(ConnEvent::close); }

static void on_error(string msg)
{
    console::log(msg);
    conn_SM->feed(ConnEvent::close);
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

// Handler for messages received from the server
static void on_message(string data, bool extracted)
{
    try {
        if (debug) {
            string s;
            s.reserve(data.size() + 3);
            if (extracted) {
                s += '\t';
            }
            s += "< " + data;
            console::log(s);
        }

        if (data.size() < 3) {
            console::error("message too short: " + data);
            return;
        }

        const Message type = static_cast<Message>(std::stoul(data));
        data = data.substr(2);
        switch (type) {
        case Message::synchronise:
            synchronize();
            break;
        case Message::concat: {
            // Split several concatenated messages
            string frag;
            frag.reserve(data.size());
            size_t last = 0;
            while (1) {
                const size_t i = frag.find(char(0), last);
                on_message(data.substr(last, i), true);
                if (i == -1) {
                    break;
                }
                last = i + 1;
            }
            return;
        }
        default:
            console::warn("unknown WS message: " + encode_message(type, data));
            return;
        }
    } catch (const std::exception& ex) {
        console::error(ex.what());
    }
}

EMSCRIPTEN_BINDINGS(module_conn)
{
    using emscripten::function;

    function("_on_socket_open", &on_open);
    function("_on_socket_close", &on_close);
    function("_on_socket_message", &on_message);
}

static void close_socket()
{
    EM_ASM({
        if (widow.__socket) {
            window.__socket.close();
            window.__socket = null;
        }
    });
}

static void connect()
{
    close_socket();
    EM_ASM({
        var path = (location.protocol == 'https:' ? 'wss' : 'ws') + '://'
            + location.host + '/api/socket';
        var s = window.__socket = new WebSocket(path);
        s.onopen = Module._on_socket_open;
        s.onclose = Module._on_socket_close;
        s.onerror = function(e)
        {
            console.error(e);
            Module._on_socket_close();
        };
        s.onmessage = function(e) { Module._on_socket_message(e.data, false); };
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

void init_connectivity()
{
    conn_SM = new FSM<ConnState, ConnEvent>(ConnState::loading);
    if (!page->thread) {
        return;
    }

    // TODO
}
