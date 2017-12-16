#include "state.hh"
#include "../state.hh"
#include "../util.hh"
#include <emscripten.h>
#include <emscripten/bind.h>

using std::string;

FSM<ConnState, ConnEvent>* conn_SM = nullptr;

static void on_open() { conn_SM->feed(ConnEvent::error); }

static void on_close() { conn_SM->feed(ConnEvent::close); }

static void on_error(string msg)
{
    console::log(msg);
    conn_SM->feed(ConnEvent::close);
}

static void on_message(string data, bool extracted)
{
    // TODO
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

void init_connection()
{
    conn_SM = new FSM<ConnState, ConnEvent>(ConnState::loading);
    if (!page->thread) {
        return;
    }

    // TODO
}

void send_message(Message type, string msg)
{
    // TODO
}
