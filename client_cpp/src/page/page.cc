#include "../../brunhild/mutations.hh"
#include "../posts/models.hh"
#include "../state.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <sstream>

void render_page()
{
    brunhild::set_inner_html("threads", "");

    std::ostringstream s;
    for (auto & [ id, p ] : *posts) {
        p.is_rendered = true;
        p.init(p.render());
        brunhild::append("threads", p.html());
    }
}

EMSCRIPTEN_BINDINGS(module_page)
{
    emscripten::function("render_page", &render_page);
}
