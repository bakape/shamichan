#include "node.hh"

unsigned long long id_counter = 0;

namespace brunhild {

std::string new_id()
{
    std::string s;
    s.reserve(16);
    s += "bh-" + std::to_string(id_counter++);
    return s;
}

std::string Node::html() const
{
    std::string s;
    s.reserve(1 << 10);
    write_html(s);
    return s;
}

void Node::write_html(std::string& s) const
{
    if (tag == "_text") {
        s += attrs.at("_text");
        return;
    }

    s += '<' + tag;
    for (auto & [ key, val ] : attrs) {
        s += ' ' + key;
        if (val != "") {
            s += "=\"" + val + '"';
        }
    }
    s += '>';

    for (auto& ch : children) {
        ch.write_html(s);
    }

    s += "</" + tag + '>';
}

Node Node::text(std::string text)
{
    return Node("_text", { { "_text", text } });
}

void Node::clear()
{
    tag.clear();
    attrs.clear();
    children.clear();
}

void escape(std::string& out, const std::string& s)
{
    for (auto ch : s) {
        switch (ch) {
        case '&':
            out += "&amp;";
            break;
        case '\"':
            out += "&quot;";
            break;
        case '\'':
            out += "&apos;";
            break;
        case '<':
            out += "&lt;";
            break;
        case '>':
            out += "&gt;";
            break;
        case '`':
            out += "&#x60;";
            break;
        default:
            out += ch;
        }
    }
}

Node Node::escaped(const std::string& s)
{
    std::string out;
    out.reserve(s.size() * 1.2);
    brunhild::escape(out, s);
    return Node::text(out);
}
}
