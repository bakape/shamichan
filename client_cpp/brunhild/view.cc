#include "view.hh"
#include "mutations.hh"
#include <sstream>

namespace brunhild {

void View::append(std::string html) { brunhild::append(id, html); }

void View::append(const Node& node) { append(node.html()); }

void View::prepend(std::string html) { brunhild::prepend(id, html); }

void View::prepend(const Node& node) { prepend(node.html()); }

void View::before(std::string html) { brunhild::before(id, html); }

void View::before(const Node& node) { before(node.html()); }

void View::after(std::string html) { brunhild::after(id, html); }

void View::after(const Node& node) { after(node.html()); }

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
}
