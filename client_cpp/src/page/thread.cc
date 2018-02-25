#include "thread.hh"
#include "../../brunhild/mutations.hh"
#include "../lang.hh"
#include "../state.hh"
#include "page.hh"
#include <ctime>
#include <optional>
#include <sstream>

using brunhild::Node;
using std::string;

void render_thread()
{
    // TODO: Disable live posting toggle in non-live threads

    const Thread& thread = threads.at(page.thread);
    std::ostringstream s;

    Node n("span", { { "class", "aside-container top-margin" } },
        {
            render_button("#bottom", lang.ui.at("bottom")),
            render_button(".", lang.ui.at("return")),
            { "span", "TODO: Catalog" },
            // render_button("catalog", lang.ui.at("catalog")),
            render_button(std::nullopt, lang.posts.at("expandImages")),
        });
    push_board_hover_info(n.children);
    n.write_html(s);

    s << "<hr><section id=\"thread-container\">";
    for (auto & [ _, p ] : posts) {
        p.init();
        p.write_html(s);
    }
    s << "</section><div id=\"bottom-spacer\"></div>";

    if (!thread.locked) {
        Node({
                 "aside",
                 { { "class", "act posting glass" } },
                 { { "a", lang.ui.at("reply") } },
             })
            .write_html(s);
    }
    s << "<hr>";

    n = Node("span", { { "class", "aside-container" }, { "id", "bottom" } });
    n.children.push_back(render_button(".", lang.ui.at("return")));
    n.children.push_back({ "span", "TODO: Catalog" });
    n.children.push_back(render_button("#top", lang.ui.at("top")));
    n.children.push_back(render_last_100_link(page.board, page.thread));
    n.children.push_back({
        "span",
        { { "id", "lock" }, { "style", "visibility: hidden;" } },
        lang.ui.at("lockedToBottom"),
    });
    n.write_html(s);

    brunhild::set_inner_html("threads", s.str());
    set_title(format_title(page.board, threads.at(page.thread).subject));
    render_post_counter();
}

void render_post_counter()
{
    std::ostringstream s;
    if (page.thread) {
        auto const& t = threads.at(page.thread);
        s << t.post_ctr << " / " << t.image_ctr;

        // Calculate estimated thread expiry time
        if (config.prune_threads) {
            // Calculate expiry age
            const auto min = config.thread_expiry_min;
            const auto max = config.thread_expiry_max;
            const double p = (double)(t.post_ctr) / 3000 - 1;
            double days = min + (-max + min) * p * p * p;
            if (t.deleted) {
                days /= 3;
            }
            if (days < min) {
                days = min;
            }

            // Subtract current bump time
            days -= (std::time(0) - t.bump_time) / (3600 * 24);

            s << " / ";
            if (days > 1) {
                s << (int)(days) << 'd';
            } else {
                s << (int)(days / 24) << 'h';
            }
        }
    }
    brunhild::set_inner_html("thread-post-counters", s.str());
}
