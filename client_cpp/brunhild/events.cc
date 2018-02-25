#include "events.hh"
#include <emscripten.h>
#include <map>
#include <unordered_map>

using std::string;

namespace brunhild {

// used to identify event handler targets
typedef std::pair<std::string, std::string> Key;

struct pairhash {
public:
    template <typename T, typename U>
    std::size_t operator()(const std::pair<T, U>& x) const
    {
        return std::hash<T>()(x.first) ^ (std::hash<U>()(x.second) << 1);
    }
};

// All registered event handlers
static std::unordered_map<Key, std::unordered_map<long, Handler>, pairhash>
    handlers;

static long id_counter = 0;

long register_handler(string type, Handler handler, string selector)
{
    const Key key = { type, selector };

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
                                if (!sel || t.matches(sel)) {
                                    Module._run_event_handlers(type, sel, e);
                                }
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
    for (auto & [ key, h_set ] : handlers) {
        for (auto[h_id, _] : h_set) {
            if (h_id == id) {
                h_set.erase(h_id);
                // Not removing global listener completely, as that would
                // require tracking handler functions
                EM_ASM_INT(
                    {
                        delete window
                            .__bh_handlers[UTF8ToString($0)][UTF8ToString($1)];
                    },
                    key.first.c_str(), key.second.c_str());
                return;
            }
        }
    }
}

static void run_event_handlers(string type, string sel, emscripten::val event)
{
    const Key key = { type, sel };
    if (!handlers.count(key)) {
        return;
    }
    for (auto & [ _, h ] : handlers.at(key)) {
        h(event);
    }
}

EMSCRIPTEN_BINDINGS(module_events)
{
    emscripten::function("_run_event_handlers", &run_event_handlers);
}
}
