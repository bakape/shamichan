#include "../lang.hh"
#include "../options/options.hh"
#include "../state.hh"
#include "../util.hh"
#include "etc.hh"
#include "view.hh"
#include <sstream>
#include <vector>

using std::optional;

// Render omitted post and image count for shortened threads by thread ID
static optional<Node> render_omitted(unsigned long id, std::string board)
{
    if (!threads.count(id)) {
        return {};
    }
    auto const& t = threads.at(id);

    // Collect all posts for this thread
    std::vector<Post*> owned;
    owned.reserve(32);
    for (auto & [ _, p ] : posts) {
        if (p.op == id) {
            owned.push_back(&p);
        }
    }

    // Calculate omitted posts and images
    long omit = long(t.post_ctr) - owned.size();
    long image_omit = 0;
    if (omit) {
        image_omit = t.image_ctr;
        for (auto p : owned) {
            if (p->image) {
                image_omit--;
            }
        }
    } else {
        return {};
    }

    std::ostringstream s;
    s << pluralize(omit, "post") << ' ' << lang.posts.at("and") << ' '
      << pluralize(image_omit, "image") << ' ' << lang.posts.at("omitted");
    return {
        {
            "span", { { "class", "omit spaced" } },
            // Disambiguate constructor
            brunhild::Children({
                { "span", s.str() },
                render_button(
                    absolute_thread_url(id, board), lang.posts.at("seeAll")),
            }),
        },
    };
}

Node PostView::render(Post* m)
{
    if (post_ids.hidden.count(m->id)) {
        return { "article", { { "hidden", "" } } };
    }

    Node n = { "article", { { "class", "glass" } } };
    n.children.reserve(4);

    if (m->id == m->op) {
        n.attrs["class"] += " op";
    }
    if (m->editing) {
        n.attrs["class"] += " editing";
    }
    if (m->deleted) {
        n.attrs["class"] += " deleted";
        n.children.push_back(delete_toggle);
    }
    n.children.push_back(render_header());

    brunhild::Children pc_ch;
    pc_ch.reserve(2);
    if (m->image) {
        n.children.push_back(render_figcaption());
        if ((!options.hide_thumbs && !options.work_mode_toggle)
            || reveal_thumbnail) {
            auto[figure, audio] = render_image();
            pc_ch.push_back(figure);

            // Will be false almost always, so need to reserve memory for this
            if (audio) {
                pc_ch.push_back(*audio);
            }
        }
    }
    pc_ch.push_back(render_body());
    if (m->banned) {
        n.children.push_back(
            { "b", { { "class", "admin banned" } }, lang.posts.at("banned") });
    }
    n.children.push_back({ "div", { { "class", "post-container" } }, pc_ch });

    if (m->id == m->op) {
        if (auto omit = render_omitted(m->id, m->board); omit) {
            n.children.push_back(*omit);
        }
    }
    if (m->backlinks.size()) {
        Node bl("span", { { "class", "backlinks" } });
        for (auto && [ id, data ] : m->backlinks) {
            bl.children.push_back(render_link(id, data));
        }
        n.children.push_back(bl);
    }

    return n;
}
