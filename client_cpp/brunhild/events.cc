#include "events.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <map>
#include <unordered_map>
#include <vector>

using std::string;
using std::unordered_map;
using std::vector;

namespace brunhild {

// All registered event handlers
unordered_map<string, vector<Handler>>* handlers = nullptr;

void register_handler(string type, Handler handler, string selector)
{
    if (!handlers) {
        handlers = new unordered_map<string, vector<Handler>>();
    }

    const string key = type + ':' + selector;

    if (!handlers->count(key)) {
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

                            var attrs; // Lazy attribute encoding
                            for (var sel in window.__bh_handlers[type]) {
                                if (sel && (!t.matches || !t.matches(sel))) {
                                    continue;
                                }
                                if (!attrs) {
                                    attrs = new Module._StringMap();
                                    var a = t.attributes;
                                    for (var i = 0; i < a.length; i++) {
                                        attrs.set(a[i].name, a[i].value);
                                    }
                                }
                                Module._run_event_handler(
                                    type + ':' + sel, t.tagName, attrs);
                            }
                        },
                        { passive : true });
                }
                window.__bh_handlers[type][sel] = true;
            },
            type.c_str(), selector.c_str());
    }

    (*handlers)[key].push_back(handler);
}

static void run_event_handler(
    string key, string tag, std::map<string, string> attrs)
{
    const EventTarget data = { tag, Attrs(attrs.begin(), attrs.end()) };
    for (auto fn : handlers->at(key)) {
        (*fn)(data);
    }
}

EMSCRIPTEN_BINDINGS(module_events)
{
    using namespace emscripten;

    function("_run_event_handler", &run_event_handler);
    register_map<string, string>("_StringMap");
}
}
