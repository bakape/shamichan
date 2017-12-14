#include "../options/options.hh"
#include "etc.hh"
#include "models.hh"

Node Post::render()
{
    Node n("article", { { "id", 'p' + std::to_string(id) } });
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

    if (backlinks.size()) {
        Node bl("span", { { "class", "backlinks" } });
        for (auto && [ id, data ] : backlinks) {
            bl.children.push_back(render_post_link(id, data));
        }
        n.children.push_back(bl);
    }

    return n;
}
