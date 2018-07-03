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

std::string HTMLWriter::html()
{
    Rope s;
    write_html(s);
    return s.str();
}

void Attrs::write_html(Rope& s)
{
    for (auto & [ key, val ] : *this) {
        s << ' ' << key;
        if (val != "") {
            s << "=\"" << val << '"';
        }
    }
}

void Attrs::patch(Attrs&& attrs)
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

void Node::write_html(Rope& s)
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

void Node::stringify_subtree()
{
    Rope s;
    for (auto& ch : children) {
        ch.write_html(s);
    }
    inner_html = s.str();
    children.clear();
}

void Node::clear()
{
    tag.clear();
    attrs.clear();
    children.clear();
    inner_html = std::nullopt;
}

void Node::hide() { attrs["hide"] = ""; }
}
