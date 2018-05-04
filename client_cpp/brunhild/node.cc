#include "node.hh"
#include "mutations.hh"
#include "util.hh"

static unsigned long long id_counter = 0;

namespace brunhild {

std::string new_id()
{
    std::ostringstream s;
    s << "bh-" << id_counter++;
    return s.str();
}

void Attrs::write_html(Rope& s) const
{
    for (auto & [ key, val ] : *this) {
        s << ' ' << key;
        if (val != "") {
            s << "=\"" << val << '"';
        }
    }
}

void Attrs::patch(Attrs attrs)
{
    const auto id = (*this)["id"];
    bool patched = false;

    // Attributes added or changed
    for (auto & [ key, val ] : attrs) {
        if (key != "id" && (!count(key) || at(key) != val)) {
            set_attr(id, key, val);
            patched = true;
        }
    }

    // Attributes removed
    for (auto & [ key, _ ] : *this) {
        if (key != "id" && !attrs.count(key)) {
            remove_attr(id, key);
            patched = true;
        }
    }

    if (patched) {
        *this = attrs;
        (*this)["id"] = id;
    }
}

std::string Node::html() const
{
    Rope s;
    write_html(s);
    return s.str();
}

void Node::write_html(Rope& s) const
{
    s << '<' << tag;
    attrs.write_html(s);
    s << '>';

    // These should be left empty and unterminated
    if (tag == "br" || tag == "wbr") {
        return;
    }

    if (inner_html) {
        s << *inner_html;
    } else {
        for (auto& ch : children) {
            ch.write_html(s);
        }
    }

    s << "</" << tag << '>';
}

std::string render_children(const Children& children)
{
    Rope s;
    for (auto& ch : children) {
        ch.write_html(s);
    }
    return s.str();
}

void Node::stringify_subtree()
{
    inner_html = render_children(children);
    children.clear();
}

void Node::clear()
{
    tag.clear();
    attrs.clear();
    children.clear();
    inner_html = std::nullopt;
}
}
