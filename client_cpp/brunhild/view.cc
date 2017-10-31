#include "view.hh"
#include "mutations.hh"
#include <sstream>

namespace brunhild {

void View::append(std::string html) { brunhild::append(id, html); }

void View::prepend(std::string html) { brunhild::prepend(id, html); }

void View::before(std::string html) { brunhild::before(id, html); }

void View::after(std::string html) { brunhild::after(id, html); }

void View::set_inner_html(std::string html)
{
    brunhild::set_inner_html(id, html);
}

void View::set_children(const Children& children)
{
    std::ostringstream s;
    for (auto& ch : children) {
        ch.write_html(s);
    }
    set_inner_html(s.str());
}

void View::remove() { brunhild::remove(id); }

void View::set_attr(std::string key, std::string val)
{
    brunhild::set_attr(id, key, val);
}

void View::remove_attr(std::string key) { brunhild::remove_attr(id, key); }

void VirtualView::init(Node node)
{
    saved = node;
    ensure_id(saved);
}

void VirtualView::ensure_id(Node& node)
{
    if (!node.attrs.count("id")) {
        node.attrs["id"] = new_id();
    }
    for (auto& ch : node.children) {
        ensure_id(ch);
    }
}

std::string VirtualView::html() const { return saved.html(); }

void VirtualView::write_html(std::ostringstream& s) const
{
    saved.write_html(s);
}

void VirtualView::patch(Node node) { patch_node(saved, node); }

void VirtualView::patch_node(Node& old, Node node)
{
    // Completely replace node and subtree
    bool replace = old.tag != node.tag;
    if (!replace) {
        if (node.attrs.count("id") && node.attrs["id"] != old.attrs["id"]) {
            replace = true;
        }
    }
    if (replace) {
        auto const old_id = old.attrs["id"];
        old = node;
        set_outer_html(old_id, old.html());
        return;
    }

    patch_attrs(old, node.attrs);
    patch_children(old, node.children);
}

void VirtualView::patch_attrs(Node& old, Attrs attrs)
{
    if (old.attrs == attrs) {
        return;
    }

    // Attributes added or changed
    for (auto && [ key, val ] : attrs) {
        if (!old.attrs.count(key) || old.attrs[key] != val) {
            set_attr(old.attrs["id"], key, val);
        }
    }

    // Attributes removed
    for (auto && [ key, _ ] : old.attrs) {
        if (!attrs.count(key)) {
            remove_attr(old.attrs["id"], key);
        }
    }

    old.attrs = attrs;
}

void VirtualView::patch_children(Node& old, Children ch)
{
    // Text nodes can not be addressed by ID and require special handling
    if (old.children.size() == 1 && old.children.front().is_text()) {
        // Hot path
        if (ch.size() == 1 && ch.front().is_text()) {
            auto const& text = ch.front().attrs["_text"];
            if (old.attrs["_text"] != text) {
                set_inner_html(old.attrs["id"], text);
                old.children = ch;
            }
            return;
        }

        std::ostringstream s;
        for (auto& ch : ch) {
            ensure_id(ch);
            ch.write_html(s);
        }
        old.children = ch;
        set_inner_html(old.attrs["id"], s.str());
        return;
    } else if (ch.size() == 1 && ch.front().is_text()) {
        set_inner_html(old.attrs["id"], ch.front().attrs["_text"]);
        old.children = ch;
        return;
    }

    int diff = ch.size() - old.children.size();

    // Remove Nodes from the end
    while (diff < 0) {
        brunhild::remove(old.children.back().attrs["id"]);
        old.children.pop_back();
        diff++;
    }

    auto old_iter = old.children.begin();
    auto ch_iter = ch.begin();
    while (old_iter != old.children.end() && ch_iter != ch.end()) {
        patch_node(*old_iter, *ch_iter);
    }

    // Append Nodes
    if (diff > 0) {
        while (ch_iter != ch.end()) {
            auto ch = *ch_iter;
            ensure_id(ch);
            append(old.attrs["id"], ch.html());
            old.children.push_back(ch);
        }
    }
}
}
