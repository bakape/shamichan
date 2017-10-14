#include "../../brunhild/mutations.hh"
#include "../state.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <sstream>
#include "../posts/view.hh"

void render_page()
{
    std::ostringstream s;
    for (auto & [ id, p ] : *posts) {
        if (p.view) {
            delete p.view;
        }
        p.view = new PostView(p);
        p.view->write_html(s);
    }
    brunhild::set_inner_html("threads", s.str());
}

EMSCRIPTEN_BINDINGS(module_page)
{
    emscripten::function("render_page", &render_page);
}
