#include "../brunhild/init.hh"
#include "../brunhild/mutations.hh"
#include "local_storage.hh"
#include "page/navigation.hh"
#include "page/scroll.hh"
#include "posts/commands.hh"
#include "posts/init.hh"
#include "state.hh"

int main()
{
    brunhild::before_flush = &rerender_syncwatches;
    brunhild::after_flush = &compensate_scrolling;
    brunhild::init();
    load_state();
    init_posts();
    init_navigation();
    init_scrolling();
    if (page->post) {
        scroll_to_post(page->post);
    }
    return 0;
}
