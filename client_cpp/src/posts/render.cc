#include "../lang.hh"
#include "../options/options.hh"
#include "../state.hh"
#include "../util.hh"
#include "etc.hh"
#include "models.hh"
#include <sstream>
#include <vector>

using std::optional;

// Render omitted post and image count for shortened threads by thread ID
static optional<Node> render_omitted(unsigned long id)
{
    if (!threads->count(id)) {
        return {};
    }
    auto const& t = threads->at(id);

    // There might still be posts missing due to deletions even in complete
    // thread queries. Ensure we are actually retrieving an abbreviated thread
    // before calculating.
    if (!t.abbrev) {
        return {};
    }

    // Collect all posts for this thread
    std::vector<Post*> owned;
    owned.reserve(32);
    for (auto & [ _, p ] : *posts) {
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
    s << pluralize(omit, "post") << ' ' << lang->posts.at("and") << ' '
      << pluralize(image_omit, "image") << ' ' << lang->posts.at("omitted");
    return {
        {
            "span",
            { { "class", "omit spaced" } },
            // Disambiguate constructor
            brunhild::Children({
                { "span", s.str() },
                render_button(std::to_string(id), lang->posts.at("seeAll")),
            }),
        },
    };
}

Node Post::render()
{
    Node n("article", { { "id", 'p' + std::to_string(id) } });
    if (post_ids->hidden.count(id)) {
        // No need to do useless work
        n.attrs["class"] = "hidden";
        return n;
    }
    n.children.reserve(4);

    n.attrs["class"] = "glass";
    if (editing) {
        n.attrs["class"] += " editing";
    }
    if (deleted) {
        n.attrs["class"] += " deleted";
        n.children.push_back(delete_toggle);
    }
    n.children.push_back(render_header());

    brunhild::Children pc_ch;
    pc_ch.reserve(2);
    if (image) {
        n.children.push_back(render_figcaption());
        if ((!options->hide_thumbs && !options->work_mode_toggle)
            || image->reveal_thumbnail) {
            auto[figure, audio] = render_image();
            pc_ch.push_back(figure);

            // Will be false almost always, so need to reserve memory for this
            if (audio) {
                pc_ch.push_back(*audio);
            }
        }
    }
    pc_ch.push_back(render_body());
    n.children.push_back({ "div", { { "class", "post-container" } }, pc_ch });

    if (id == op) {
        if (auto omit = render_omitted(id); omit) {
            n.children.push_back(*omit);
        }
    }
    if (backlinks.size()) {
        Node bl("span", { { "class", "backlinks" } });
        for (auto && [ id, data ] : backlinks) {
            bl.children.push_back(render_post_link(id, data));
        }
        n.children.push_back(bl);
    }

    return n;
}
