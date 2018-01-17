#include "http.hh"
#include "util.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <map>

// All registered callbacks
std::map<unsigned, HTTPCallback> callbacks;

// Last ID used
unsigned last_id = 0;

void http_request(std::string url, HTTPCallback cb)
{
    const unsigned id = last_id++;
    callbacks[id] = cb;
    EM_ASM_INT(
        {
            var xhr = new XMHLHTTPRequest();
            xhr.open(UTF8ToString($0), "GET");
            xhr.onload = function()
            {
                Module.run_http_callback($1, xhr.status, xhr.response);
            };
            xhr.send();
        },
        url.c_str(), id);
}

static void run_http_callback(
    unsigned id, unsigned short code, std::string data)
{
    callbacks.at(id)(code, data);
    callbacks.erase(id);
}

EMSCRIPTEN_BINDINGS(module_http)
{
    emscripten::function("run_HTTP_callback", &run_http_callback);
}
