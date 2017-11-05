#include "../../brunhild/mutations.hh"
#include "../posts/models.hh"
#include "../state.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <sstream>

void render_page()
{
    std::ostringstream s;
    for (auto & [ id, p ] : *posts) {
        p.is_rendered = true;
        p.init(p.render());
        p.write_html(s);
    }
    brunhild::set_inner_html("threads", s.str());
}

EMSCRIPTEN_BINDINGS(module_page)
{
    emscripten::function("render_page", &render_page);
}
