#include "node.hh"

unsigned long long id_counter = 0;

namespace brunhild {

std::string new_id()
{
    std::ostringstream s;
    s << "bh-" << id_counter++;
    return s.str();
}

std::string Node::html() const
{
    std::ostringstream s;
    write_html(s);
    return s.str();
}

void Node::write_html(std::ostringstream& s) const
{
    if (tag == "_text") {
        s << attrs.at("_text");
        return;
    }

    s << '<' << tag;
    for (auto & [ key, val ] : attrs) {
        s << ' ' << key;
        if (val != "") {
            s << "=\"" << val << '"';
        }
    }
    s << '>';

    for (auto& ch : children) {
        ch.write_html(s);
    }

    s << "</" << tag << '>';
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

std::string escape(const std::string& s)
{
    std::string out;
    out.reserve(s.size() * 1.2);
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
    return out;
}

Node Node::escaped(const std::string& s)
{
    return Node::text(brunhild::escape(s));
}
}
