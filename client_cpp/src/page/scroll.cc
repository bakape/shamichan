#include "scroll.hh"
#include "../../brunhild/mutations.hh"
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
    brunhild::scroll_into_view('p' + std::to_string(id));
    scroll_by = -top_banner_height;
}
