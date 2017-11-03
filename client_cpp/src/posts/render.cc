#include "../options/options.hh"
#include "models.hh"
#include "util.hh"

Node Post::render()
{
    Node n = { "article", { { "id", "#p" + std::to_string(id) } } };

    n.attrs["class"] = "glass";
    if (editing) {
        n.attrs["class"] += " editing";
    }

    if (deleted) {
        n.attrs["class"] += " deleted";
        n.children.push_back({
            "input", { { "type", "checkbox" }, { "class", "deleted-toggle" } },
        });
    }
    n.children.push_back(render_header());

    brunhild::Children pc_ch;
    if (image) {
        n.attrs["class"] += " media";
        n.children.push_back(render_figcaption());
        if ((!options->hide_thumbs && !options->work_mode_toggle)
            || image->reveal_thumbnail) {
            pc_ch.push_back(render_image());
        }
    }
    pc_ch.push_back(render_body());
    n.children.push_back({ "div", { { "class", "post-container" } }, pc_ch });

    if (backlinks.size()) {
        Node bl("span", { { "class", "backlinks" } });
        for (auto && [ id, data ] : backlinks) {
            bl.children.push_back(render_post_link(id, data));
        }
        n.children.push_back(bl);
    }

    return n;
}
