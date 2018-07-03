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
    brunhild::Rope s;

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

    s << "<hr>";

    // TODO: Store this somewhere
    auto tv = new ThreadView(page.thread, "thread-container");
    tv->write_html(s);
    s << "<div id=\"bottom-spacer\"></div>";

    if (!thread.locked) {
        Node({
                 "aside", { { "class", "act posting glass" } },
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
        "span", { { "id", "lock" }, { "style", "visibility: hidden;" } },
        lang.ui.at("lockedToBottom"),
    });
    n.write_html(s);

    brunhild::set_inner_html("threads", s.str());
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

ThreadView::ThreadView(unsigned long thread_id, std::string id)
    : ListView("section", id)
    , thread_id(thread_id)
{
    ThreadView::instances[thread_id] = this;
}

std::vector<Post*> ThreadView::get_list()
{
    std::vector<Post*> re;
    re.reserve(1 << 9);
    for (auto & [ _, p ] : posts) {
        if (p.op == thread_id) {
            re.push_back(&posts.at(p.id));
        }
    }
    return re;
}

std::shared_ptr<PostView> ThreadView::create_child(Post* p)
{
    return p->views.emplace_back(new PostView(p->id));
}

std::vector<brunhild::View*> ThreadPageView::top_controls()
{
    return { new Button(lang.ui.at("bottom"), "#bottom"),
        new Button(lang.ui.at("return"), "."),
        new Button(lang.ui.at("catalog"), "catalog") };
}

std::vector<brunhild::View*> ThreadPageView::bottom_controls()
{
    return { new Button(lang.ui.at("top"), "#top"),
        new Button(lang.ui.at("return"), "."),
        new Button(lang.ui.at("catalog"), "catalog"),
        new brunhild::NodeView({
            "span", { { "id", "lock" }, { "style", "visibility: hidden;" } },
            lang.ui.at("lockedToBottom"),
        }) };
}
