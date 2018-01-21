#include "connection.hh"
#include "../../brunhild/mutations.hh"
#include "../../utf8/utf8.h"
#include "../json.hh"
#include "../lang.hh"
#include "../page/thread.hh"
#include "../state.hh"
#include "../util.hh"
#include "posts.hh"
#include "sync.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <functional>
#include <iterator>

using nlohmann::json;
using std::string;

FSM<ConnState, ConnEvent>* conn_SM = nullptr;

static void on_open() { conn_SM->feed(ConnEvent::open); }

static void on_close() { conn_SM->feed(ConnEvent::close); }

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

// Run handler on post, if post exists on page.
// A post may not be loaded in this client but still exist in the
// thread.
static void if_post_exists(
    const unsigned long id, std::function<void(Post&)> fn)
{
    if (posts->count(id)) {
        auto& p = posts->at(id);
        fn(p);
    }
}

// Same for posts with JSON object messages and an "id" attribute, that defines
// the post ID
static void if_post_exists(
    std::string_view data, std::function<void(json&, Post&)> fn)
{
    auto j = json::parse(data);
    if_post_exists(j["id"].get<unsigned long>(), [&](auto& p) { fn(j, p); });
}

// Mimics JS Array.splice() method for inserting and removing text from UTF-8
// strings at a certain position, but in place
static void splice(string& s, int start, int len, string text)
{
    auto start_pos = s.begin();
    utf8::advance(start_pos, start, s.end());
    auto end_pos = start_pos;
    utf8::advance(end_pos, len, s.end());
    string end(end_pos, s.end());
    s = s.substr(0, start_pos - s.begin());
    s.reserve(s.size() + text.size() + end.size());
    s += text;
    s += end;
}

// Set synced IP count to n
static void render_sync_count(unsigned n)
{
    string s;
    if (n) {
        s = std::to_string(n);
    }
    brunhild::set_inner_html("sync-counter", s);
}

// Handler for messages received from the server.
// extracted specifies, the mesage was extracted from a larger concatenated
// message.
static void on_message(std::string_view msg, bool extracted)
{
    if (debug) {
        string s;
        s.reserve(msg.size() + 3);
        if (extracted) {
            s += '\t';
        }
        s += "< " + string(msg);
        console::log(s);
    }

    const Message type
        = static_cast<Message>(std::stoul(string(msg.substr(0, 2))));

    // Guard against messages possibly resulted from rapid changing of feeds and
    // high latency
    if (conn_SM->state() != ConnState::synced) {
        switch (type) {
        case Message::invalid:
        case Message::synchronise:
            break;
        default:
            return;
        }
    }

    auto data = msg.substr(2);
    switch (type) {
    case Message::invalid:
        alert(string(data));
        conn_SM->feed(ConnEvent::error);
        break;
    case Message::insert_post:
        insert_post(data);
        break;
    case Message::insert_image:
        if_post_exists(data, [](auto& j, auto& p) {
            p.image = Image(j);
            p.patch();
            threads->at(page->thread).image_ctr++;
            render_post_counter();

            // TODO: Image auto expansion

        });
        break;
    case Message::spoiler:
        if_post_exists(std::stoul(string(data)), [](auto& p) {
            p.image->spoiler = true;
            p.patch();
        });
        break;
    case Message::append: {
        auto j = json::parse(data);
        if_post_exists(j[0].get<unsigned long>(), [&](auto& p) {
            utf8::unchecked::append(j[1], std::back_inserter(p.body));
            p.patch();
        });
    } break;
    case Message::backspace:
        if_post_exists(std::stoul(string(data)), [](auto& p) {
            // Removes the last UTF-8 char from the post's text
            auto it = p.body.end();
            utf8::unchecked::prior(it);
            p.body = p.body.substr(0, it - p.body.begin());
            p.patch();
        });
        break;
    case Message::splice:
        if_post_exists(data, [](auto& j, auto& p) {
            splice(p.body, j["start"], j["len"], j["text"]);
            p.patch();
        });
        break;
    case Message::close_post:
        if_post_exists(data, [](auto& j, auto& p) {
            if (j.count("links")) {
                p.parse_links(j);
                p.propagate_links();
            }
            p.parse_commands(j);
            p.close();
        });
        break;
    case Message::synchronise:
        load_posts(data);
        conn_SM->feed(ConnEvent::sync);
        break;
    case Message::sync_count:
        render_sync_count(std::stoul(string(data)));
        break;
    case Message::concat: {
        // Split several concatenated messages
        string s;
        for (auto& msg : json::parse(data)) {
            s = msg;
            on_message(std::string_view(s), true);
        }
        return;
    }
    default:
        console::warn(
            "unknown websocket message: " + encode_message(type, string(data)));
        return;
    }
}

// Takes a raw char* as int.
// Not using embind here, because it does not support UTF-8. This also reduces
// copying.
static void on_message_raw(int msg_ptr)
{
    // Binding to a variable keeps the underlying char* from dealocating till
    // scope exit
    auto v = c_string_view((char*)(msg_ptr));
    on_message(v.substr(), false);
}

static void retry_to_connect() { conn_SM->feed(ConnEvent::retry); }

// Work around browser slowing down/suspending tabs and keep the FSM up to
// date with the actual status.
static void resync_conn_SM()
{
    switch (conn_SM->state()) {
    // Ensure still connected, in case the computer went to sleep or
    // hibernate or the mobile browser tab was suspended.
    case ConnState::synced:
        send_message(Message::NOP, "");
        break;
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
    function("on_socket_message", &on_message_raw);
    function("retry_to_connect", &retry_to_connect);
    function("resync_conn_SM", &resync_conn_SM);
}

static void connect()
{
    EM_ASM({
        if (window.__socket) {
            window.__socket.close();
        }
        var path = (location.protocol == 'https:' ? 'wss' : 'ws') + '://'
            + location.host + '/api/socket';
        var s = window.__socket = new WebSocket(path);
        s.onopen = function() { Module.on_socket_open(); };
        s.onclose = function() { Module.on_socket_close(); };
        s.onmessage = function(e)
        {
            var s = e.data;
            var len = lengthBytesUTF8(s) + 1;
            var buf = Module._malloc(len);
            stringToUTF8(s, buf, len);
            Module.on_socket_message(buf);
        };
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

// Schedule an attempt to reconnect after a connection loss
static void schedule_reconnect()
{
    EM_ASM({
        // Wait maxes out at ~1min
        var wait
            = Math.min(Math.floor(++window.__connection_attempt_count / 2), 12);
        wait = 500 * Math.pow(1.5, wait);
        setTimeout(Module.retry_to_connect, wait);
    });
}

void init_connectivity()
{
    conn_SM = new FSM<ConnState, ConnEvent>(ConnState::loading);

    // Define some JS-side functions and listeners
    EM_ASM({
        document.addEventListener('visibilitychange', function() {
            if (!document.hidden && navigator.onLine) {
                Module.resync_conn_SM();
            }
        });
        window.addEventListener(
            'online', function() { Module.retry_to_connect(); });
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
    conn_SM->act(ConnState::connecting, ConnEvent::open, []() {
        render_status(SyncStatus::connecting);
        send_sync_request();
        return ConnState::syncing;
    });
    conn_SM->act(ConnState::syncing, ConnEvent::sync, []() {
        render_status(SyncStatus::synced);
        return ConnState::synced;
    });

    // Switching from one update feed to another
    conn_SM->act(ConnState::synced, ConnEvent::switch_sync,
        []() { return ConnState::syncing; });

    conn_SM->wild_act(ConnEvent::close, []() {
        render_status(SyncStatus::disconnected);
        return conn_SM->state() == ConnState::desynced ? ConnState::desynced
                                                       : ConnState::dropped;
    });

    // schedule_reconnect() is called even on a dropped -> dropped "transition",
    // so this acts as a scheduler for new attempts
    conn_SM->on(ConnState::dropped, schedule_reconnect);
    conn_SM->act(ConnState::dropped, ConnEvent::retry, []() {
        if (!emscripten::val::global("navigator")["onLine"].as<bool>()) {
            schedule_reconnect();
            return ConnState::dropped;
        }
        connect();
        render_status(SyncStatus::connecting);
        return ConnState::connecting;
    });

    conn_SM->wild_act(ConnEvent::error, []() {
        render_status(SyncStatus::desynced);
        return ConnState::desynced;
    });
}
