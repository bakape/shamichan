#include "../brunhild/init.hh"
#include "../brunhild/mutations.hh"
#include "connection/connection.hh"
#include "db.hh"
#include "local_storage.hh"
#include "page/header.hh"
#include "page/navigation.hh"
#include "page/page.hh"
#include "posts/commands.hh"
#include "posts/init.hh"
#include "state.hh"
#include <emscripten.h>

static void start()
{
    init_connectivity();
    auto wg = new WaitGroup(2, []() {
        auto wg = new WaitGroup(1, &render_page);
        load_post_ids(wg);
    });
    open_db(wg);
    conn_SM.feed(ConnEvent::start);
    conn_SM.once(ConnState::synced, [=]() { wg->done(); });
}

int main()
{
    brunhild::before_flush = &rerender_syncwatches;
    brunhild::init();
    load_state();
    init_posts();
    init_navigation();
    brunhild::prepend("banner", board_navigation_view.html());

    start();

    // Block all clicks on <a> from exhibiting browser default behavior, unless
    // the user intends to navigate to a new tab or open a browser menu.
    // Also block navigation on form sumbition.
    EM_ASM({
        document.addEventListener('click', function(e) {
            if (e.which != 1 || e.ctrlKey) {
                return;
            }
            var t = e.target;
            switch (t.tagName) {
            case 'A':
                if (t.getAttribute('target') == '_blank'
                    || t.getAttribute('download')) {
                    return;
                }
            case 'IMG':
                e.preventDefault();
            }
        });
        document.addEventListener(
            'submit', function(e) { e.preventDefault(); });
    });

    return 0;
}
