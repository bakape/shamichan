#include "../../brunhild/mutations.hh"
#include "../posts/view.hh"
#include "../state.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <memory>
#include <sstream>

void render_page()
{
    std::ostringstream s;
    for (auto & [ id, p ] : *posts) {
        p.view = new PostView(p);
        p.view->write_html(s);
    }
    brunhild::set_inner_html("threads", s.str());
}

EMSCRIPTEN_BINDINGS(module_page)
{
    emscripten::function("render_page", &render_page);
}
