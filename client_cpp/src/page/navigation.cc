#include "../connection/connection.hh"
#include "../connection/sync.hh"
#include "../db.hh"
#include "../page/page.hh"
#include "../page/thread.hh"
#include "../state.hh"
#include "../util.hh"
#include "scroll.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <string>

void init_navigation()
{
    EM_ASM({
        history.scrollRestoration = 'manual';

        window.onpopstate = function(e)
        {
            var loc = e.target.location;
            Module.try_navigate_page(loc.href.slice(loc.origin.length), false);
        };

        document.addEventListener("click",
            function(e) {
                var t = e.target;
                if (e.which != 1 || e.ctrlKey || t.tagName != 'A'
                    || t.getAttribute('target') == '_blank'
                    || t.getAttribute('download')
                    || !t.href.startsWith(location.origin)) {
                    return;
                }
                switch (t.getAttribute('href')) {
                case '#bottom':
                case '#top':
                    location.hash = t.getAttribute('href');
                    return;
                }
                if (t.classList.contains('post-link')) {
                    if (t.classList.contains("strikethrough")
                        || localStorage.getItem('postInlineExpand') == 'true') {
                        return;
                    }
                }

                Module.try_navigate_page(
                    t.href.slice(location.origin.length), true);
            },
            { passive : true });
    });
}

// Determine, if href points to a resource on the.
// Need push signifies history.pushState() needs to be called.
static void try_navigate_page(std::string href, bool need_push)
{
    if (conn_SM.state() != ConnState::synced) {
        return;
    }

    Page next_state(href);

    // Does the link point to the same page as this one?
    const bool same_page = next_state.catalog == page.catalog
        && next_state.last_100 == page.last_100 && next_state.page == page.page
        && next_state.thread == page.thread && next_state.board == page.board;
    if (same_page) {
        if (!posts.count(next_state.post)) {
            return;
        }
        scroll_to_post(next_state.post);
        if (need_push) {
            EM_ASM_INT({ location.hash = '#p' + $0; }, next_state.post);
        }
        page = next_state;
        return;
    }

    // TODO: Reset postform
    page = next_state;
    posts.clear();
    threads.clear();
    ThreadView::clear();

    // TODO: Fetch new board configs, if needed (maybe send these in sync
    // message, if board config hash changed?)

    // TODO: New server configuration propagation. Need hash comparison on
    // server.

    // TODO: Display loading animation

    auto wg
        = new WaitGroup(2, [ full_href = location_origin + href, need_push ]() {
              render_page();
              if (need_push) {
                  EM_ASM({ history.pushState(null, null, UTF8ToString($0)); },
                      full_href.c_str());
              }
          });
    load_post_ids(wg);
    conn_SM.feed(ConnEvent::switch_sync);
    conn_SM.once(ConnState::synced, [=]() { wg->done(); });
}

EMSCRIPTEN_BINDINGS(module_navigation)
{
    emscripten::function("try_navigate_page", &try_navigate_page);
}
