#include "view.hh"
#include "../src/util.hh"
#include "mutations.hh"
#include <emscripten.h>
#include <sstream>
#include <utility>

using std::move;
using std::string;

namespace brunhild {

View::View(const string& parent_id, Node node, InsertionMode mode)
    : id(node.attrs.count("id") ? node.attrs["id"] : new_id())
{
    // In case the ID was automatically generated
    node.attrs["id"] = id;

    const auto html = node.html();
    switch (mode) {
    case InsertionMode::append:
        brunhild::append(parent_id, html);
        break;
    case InsertionMode::prepend:
        brunhild::prepend(parent_id, html);
        break;
    case InsertionMode::before:
        brunhild::before(parent_id, html);
        break;
    case InsertionMode::after:
        brunhild::after(parent_id, html);
        break;
    }

    for (auto & [ filter, fn ] : event_handlers) {
        event_handler_ids.push_back(
            register_handler(filter.type, fn, filter.selector));
    }
}

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
    const auto replace = old.tag != node.tag
        || (node.attrs.count("id")
               && node.attrs.at("id") != old.attrs.at("id"));
    if (replace) {
        const auto old_id = old.attrs.at("id");
        old = move(node);
        ensure_id(old);
        set_outer_html(old_id, old.html());
        return;
    }

    patch_attrs(old, move(node.attrs));
    patch_children(old, move(node));
}

void VirtualView::patch_attrs(Node& old, Attrs attrs)
{
    // Attributes added or changed
    for (auto & [ key, val ] : attrs) {
        if (key != "id" && (!old.attrs.count(key) || old.attrs[key] != val)) {
            old.attrs[key] = val;
            set_attr(old.attrs.at("id"), key, val);
        }
    }

    // Attributes removed
    for (auto & [ key, _ ] : old.attrs) {
        if (key != "id" && !attrs.count(key)) {
            old.attrs.erase(key);
            remove_attr(old.attrs.at("id"), key);
        }
    }
}

void VirtualView::patch_children(Node& old, Node node)
{
    // HTML string contents can not be addressed by ID and require special
    // handling
    if (old.inner_html) {
        // Hot path
        if (node.inner_html) {
            if (*old.inner_html != *node.inner_html) {
                set_inner_html(old.attrs.at("id"), *node.inner_html);
                old.inner_html = move(node.inner_html);
            }
            return;
        }

        std::ostringstream s;
        for (auto& ch : node.children) {
            ensure_id(ch);
            ch.write_html(s);
        }
        old.children = move(node.children);
        old.inner_html = std::nullopt;
        set_inner_html(old.attrs.at("id"), s.str());
        return;
    } else if (node.inner_html) {
        set_inner_html(old.attrs.at("id"), *node.inner_html);
        old.children.clear();
        old.inner_html = move(node.inner_html);
        return;
    }

    // Diff existing nodes
    for (size_t i = 0; i < old.children.size() && i < node.children.size();
         i++) {
        patch_node(old.children[i], move(node.children[i]));
    }

    int diff = int(node.children.size()) - int(old.children.size());
    if (diff > 0) { // Append Nodes
        size_t i = old.children.size();
        while (i < node.children.size()) {
            auto& ch = node.children[i++];
            ensure_id(ch);
            append(old.attrs.at("id"), ch.html());
            old.children.push_back(move(ch));
        }
    } else { // Remove Nodes from the end
        while (diff++ < 0) {
            brunhild::remove(old.children.back().attrs.at("id"));
            old.children.pop_back();
        }
    }
}
}
