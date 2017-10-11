#include "node.hh"

unsigned long long id_counter = 0;

namespace brunhild {

std::string new_id()
{
    std::ostringstream s;
    s << "bh-" << id_counter++;
    return s.str();
}

std::string Node::html()
{
    std::ostringstream s;
    write_html(s);
    return s.str();
}

void Node::write_html(std::ostringstream& s)
{
    if (tag == "_text") {
        s << attrs["_text"];
        return;
    }

    s << '<' << tag;
    for (auto & [ key, val ] : attrs) {
        s << " \"" << key << '"';
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
}
