#include "events.hh"
#include <emscripten.h>
#include <map>
#include <unordered_map>

using std::string;

namespace brunhild {
// All registered event handlers
std::unordered_map<string, std::unordered_map<long, Handler>> handlers;

long id_counter = 0;

long register_handler(string type, Handler handler, string selector)
{
    const string key = type + ':' + selector;

    if (!handlers.count(key)) {
        EM_ASM_INT(
            {
                var type = UTF8ToString($0);
                var sel = UTF8ToString($1);

                if (!window.__bh_handlers) {
                    window.__bh_handlers = {};
                }

                // Pool event handlers of one event type together
                if (!window.__bh_handlers[type]) {
                    window.__bh_handlers[type] = {};
                    document.addEventListener(type,
                        function(e) {
                            var t = e.target;
                            if (!t.tagName) { // Not an element
                                return;
                            }

                            for (var sel in window.__bh_handlers[type]) {
                                if (sel && (!t.matches || !t.matches(sel))) {
                                    continue;
                                }
                                Module._run_event_handlers(type + ':' + sel, e);
                            }
                        },
                        { passive : true });
                }
                window.__bh_handlers[type][sel] = true;
            },
            type.c_str(), selector.c_str());
    }

    const long id = id_counter++;
    handlers[key][id] = handler;
    return id;
}

void unregister_handler(long id)
{
    for (auto & [ _, h_set ] : handlers) {
        for (auto[h_id, _] : h_set) {
            if (h_id == id) {
                h_set.erase(h_id);
                return;
            }
        }
    }
}

static void run_event_handlers(string key, emscripten::val event)
{
    for (auto & [ _, h ] : handlers.at(key)) {
        (*h)(event);
    }
}

EMSCRIPTEN_BINDINGS(module_events)
{
    emscripten::function("_run_event_handlers", &run_event_handlers);
}
}
