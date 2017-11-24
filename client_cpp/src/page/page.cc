#include "../../brunhild/mutations.hh"
#include "../posts/models.hh"
#include "../state.hh"
#include "../util.hh"
#include "thread.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <sstream>

void render_page()
{
    try {
        if (page->thread) {
            render_thread();
        }
    } catch (const std::exception& ex) {
        console::error(ex.what());
    }
}

EMSCRIPTEN_BINDINGS(module_page)
{
    emscripten::function("render_page", &render_page);
}
