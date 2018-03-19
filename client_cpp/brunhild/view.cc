#include "view.hh"
#include "../src/util.hh"
#include "events.hh"
#include "mutations.hh"
#include <algorithm>
#include <emscripten.h>
#include <emscripten/bind.h>
#include <sstream>
#include <string_view>
#include <utility>

using std::move;
using std::string;

namespace brunhild {

void View::init()
{
    saved = render();
    ensure_id(saved);
}

void View::ensure_id(Node& node)
{
    if (!node.attrs.count("id")) {
        node.attrs["id"] = new_id();
    }
    for (auto& ch : node.children) {
        ensure_id(ch);
    }
}

std::string View::html() const { return saved.html(); }

void View::write_html(std::ostringstream& s) const { saved.write_html(s); }

void View::remove()
{
    brunhild::remove(saved.attrs["id"]);
    remove_event_handlers();
}

void View::patch() { patch_node(saved, render()); }

void View::patch_node(Node& old, Node node)
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

void View::patch_attrs(Node& old, Attrs attrs)
{
    const std::string id = old.attrs.at("id");

    // Attributes added or changed
    for (auto & [ key, val ] : attrs) {
        if (key != "id" && (!old.attrs.count(key) || old.attrs[key] != val)) {
            set_attr(id, key, val);
        }
    }

    // Attributes removed
    std::vector<std::string> to_remove;
    for (auto & [ key, _ ] : old.attrs) {
        if (key != "id" && !attrs.count(key)) {
            remove_attr(id, key);
        }
    }

    old.attrs = attrs;
    old.attrs["id"] = id;
}

void View::patch_children(Node& old, Node node)
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

void View::on(std::string type, std::string selector, Handler handler)
{
    // Need to prepend root node ID to all selectors
    std::ostringstream s;
    if (const string id_str = id(); selector != "") {
        std::string_view view = { selector };
        size_t i;
        while (1) {
            i = view.find(',');
            auto frag = view.substr(0, i);

            // If this comma is inside a selector like :not(.foo,.bar), skip the
            // appropriate amount of closing brackets. Assumes correct CSS
            // syntax.
            const auto opening = std::count(frag.begin(), frag.end(), '(');
            if (opening) {
                const auto closing = std::count(frag.begin(), frag.end(), ')');
                if (closing != opening) {
                    i = view.find(',', view.find(")", i, closing - opening));
                    frag = view.substr(0, i);
                }
            }

            s << '#' << id_str << ' ' << frag;
            if (i != std::string::npos) {
                view = view.substr(i + 1);
                s << ',';
            } else {
                break;
            }
        }
    } else {
        // Select all children, if no selector
        s << '#' << id_str << " *";
    }

    event_handlers.push_back(register_handler(type, handler, s.str()));
}

void View::remove_event_handlers()
{
    for (auto id : event_handlers) {
        unregister_handler(id);
    }
    event_handlers.clear();
}
}
