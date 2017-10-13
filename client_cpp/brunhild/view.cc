#include "view.hh"
#include "mutations.hh"
#include <algorithm>
#include <sstream>
#include <vector>

namespace brunhild {

void View::append(std::string html) { brunhild::append(id, html); }

void View::prepend(std::string html) { brunhild::prepend(id, html); }

void View::before(std::string html) { brunhild::before(id, html); }

void View::after(std::string html) { brunhild::after(id, html); }

void View::set_inner_html(std::string html)
{
    brunhild::set_inner_html(id, html);
}

void View::set_children(const std::vector<Node>& children)
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

VirtualView::VirtualView(Node node)
    : saved(node)
{
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

void VirtualView::patch_children(Node& old, std::vector<Node> children)
{
    int diff = children.size() - old.children.size();

    // Remove Nodes from the end
    while (diff < 0) {
        brunhild::remove(old.children.back().attrs["id"]);
        old.children.pop_back();
        diff++;
    }

    for (int i = 0; i < old.children.size(); i++) {
        patch_node(old.children[i], children[i]);
    }

    // Append Nodes
    if (diff > 0) {
        for (int i = old.children.size(); i < diff; i++) {
            auto ch = children[i];
            ensure_id(ch);
            append(old.attrs["id"], ch.html());
            old.children.push_back(ch);
        }
    }
}
}
