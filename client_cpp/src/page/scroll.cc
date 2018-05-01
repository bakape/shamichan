#include "scroll.hh"
#include "../../brunhild/mutations.hh"
#include "../posts/view.hh"
#include "../state.hh"
#include <emscripten.h>
#include <string>

int scroll_by = 0;
static unsigned top_banner_height = 0;

void compensate_scrolling()
{
    if (scroll_by) {
        EM_ASM_INT({ window.scrollBy(0, $0); }, scroll_by);
        scroll_by = 0;
    }
}

void init_scrolling()
{
    // +3 for a nicer margin
    top_banner_height = 3 + EM_ASM_INT({
        return document.getElementById('banner').offsetHeight;
    });
}

void scroll_to_post(unsigned id)
{
    if (!posts.count(id)) {
        return;
    }
    auto& p = posts.at(id);
    if (!p.views.size()) {
        return;
    }
    brunhild::scroll_into_view((*p.views.begin())->id);
    scroll_by = -top_banner_height;
}
