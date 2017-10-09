// Exposes functions to JS through Embind

#include "db.hh"
#include "page/main.hh"
#include "state.hh"
#include <emscripten/bind.h>

EMSCRIPTEN_BINDINGS(module)
{
    emscripten::function("handle_db_error", &handle_db_error);
    emscripten::function("db_is_ready", &db_is_ready);
    emscripten::function("render_page", &render_page);

    emscripten::register_vector<unsigned long>("VectorUint64");
    emscripten::function("add_to_storage", &add_to_storage);
}
