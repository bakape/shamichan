#include "../../brunhild/mutations.hh"
#include "../posts/view.hh"
#include "../state.hh"
#include <emscripten.h>
#include <emscripten/bind.h>

void render_page()
{
    std::string s;
    s.reserve(10 << 10);
    for (auto & [ id, p ] : *posts) {
        p.view = new PostView(p);
        p.view->write_html(s);
    }
    brunhild::set_inner_html("threads", s);
}

EMSCRIPTEN_BINDINGS(module_page)
{
    emscripten::function("render_page", &render_page);
}
