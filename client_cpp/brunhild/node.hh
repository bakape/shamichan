#pragma once

#include "view.hh"
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>

namespace brunhild {

// Element attributes. "" values are omitted from rendered the HTML.
typedef std::unordered_map<std::string, std::string> Attrs;

// Represents an HTML element. Can be used to construct node tries more easily.
class Node {
public:
    std::string tag;
    Attrs attrs;
    std::vector<Node> children;

    // Renders Node and subtree to HTML
    std::string html() const;

    // Creates a Node with optional attributes and children
    Node(std::string tag, Attrs attrs = {}, std::vector<Node> children = {})
        : tag(tag)
        , attrs(attrs)
        , children(children)
    {
    }

    // Creates a text Node. This node can only be a child of another Node.
    static Node text(std::string);

private:
    // Write node as HTML to stream
    friend class View;
    void write_html(std::ostringstream&) const;
};

// Generate a new unique element ID
std::string new_id();
}
