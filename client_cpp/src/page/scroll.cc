#include "scroll.hh"
#include "../posts/view.hh"
#include "../state.hh"
#include <emscripten.h>
#include <string>

void scroll_to_post(unsigned id)
{
    if (!posts.count(id)) {
        return;
    }
    auto& p = posts.at(id);
    if (!p.views.size()) {
        return;
    }
    if (auto v = brunhild::BaseView::get<PostView>(*p.views.begin()); v) {
        brunhild::scroll_into_view(v->id);
    }
}
