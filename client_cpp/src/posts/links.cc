#include "../../brunhild/node.hh"
#include "../lang.hh"
#include "../options/options.hh"
#include "../state.hh"
#include "view.hh"
#include <sstream>
#include <string>

using brunhild::Node;

Node render_post_link(uint64_t id, const LinkData& data)
{
    const bool cross_thread = data.op != page->thread;
    const bool index_page = !page->thread && !page->catalog;
    const std::string id_str = std::to_string(id);

    std::ostringstream url;
    if (cross_thread || index_page) {
        url << "/all/" << id_str;
    }
    url << "#p" << id_str;

    std::ostringstream text;
    text << ">>" << id_str;
    if (cross_thread && !index_page) {
        text << " ➡";
    }
    if (post_ids->mine.count(id)) { // Post, the user made
        text << ' ' << lang->posts.at("you");
    }

    Node n = Node("em");
    n.children.reserve(3);
    n.children.push_back({
        "a",
        {
            { "class", "post-link" }, { "href", url.str() },
        },
        text.str(),
    });
    if (options->post_inline_expand) {
        n.children.push_back({
            "a",
            {
                { "class", "hash-link" }, { "href", url.str() },
            },
            " #",
        });
    }

    // Inline linked-to post
    if (data.is_inlined) {
        auto& model = posts->at(id);
        if (!model.view) {
            model.view = new PostView();
        }
        n.children.push_back(model.view->render(model));
    }

    return n;
}
