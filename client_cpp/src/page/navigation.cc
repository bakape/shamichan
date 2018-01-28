#include "../connection/connection.hh"
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

                Module.try_navigate_page(t.href.slice(location.origin.length));
            },
            { passive : true });
    });
}

// Determine, if href points to a resource on the
static void try_navigate_page(std::string href)
{
    if (conn_SM->state() != ConnState::synced) {
        return;
    }

    Page next_state(href);

    // Does the link point to the same page as this one?
    const bool same_page = next_state.catalog == page->catalog
        && next_state.last_100 == page->last_100
        && next_state.page == page->page && next_state.thread == page->thread
        && next_state.board == page->board;
    if (same_page) {
        if (!posts->count(next_state.post)) {
            return;
        }
        scroll_to_post(next_state.post);
        EM_ASM_INT({ location.hash = '#p' + $0; }, next_state.post);
        delete page;
        page = new Page(next_state);
        return;
    }

    // TODO: Toggle loading animation
}

EMSCRIPTEN_BINDINGS(module_navigation)
{
    emscripten::function("try_navigate_page", &try_navigate_page);
}
