#pragma once

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

    // Creates a Node with optional attributes and children
    Node(std::string tag, Attrs attrs = {}, std::vector<Node> children = {})
        : tag(tag)
        , attrs(attrs)
        , children(children)
    {
    }

    // Creates a Node with attributes and a text node or subtree in the form of
    // an HTML string as the only child.
    // Escaped specifies, if the text should be escaped.
    Node(std::string tag, Attrs attrs, std::string text, bool escape = false)
        : tag(tag)
        , attrs(attrs)
        , children({ escape ? Node::escaped(text) : Node::text(text) })
    {
    }

    // Creates a Node with a text node or subtree in the form of an HTML string
    // as the only child.
    // Escaped specifies, if the text should be escaped.
    Node(std::string tag, std::string text, bool escape = false)
        : tag(tag)
        , attrs()
        , children({ escape ? Node::escaped(text) : Node::text(text) })
    {
    }

    Node() = default;

    // Renders Node and subtree to HTML
    std::string html() const;

    // Write node as HTML to stream
    void write_html(std::ostringstream&) const;

    // Resets the node and frees up used resources
    void clear();

    // Returns, if node is a text node
    bool is_text() { return tag == "_text"; }

private:
    // Creates a text Node. This node can only be a child of another Node and
    // must be the only child.
    static Node text(std::string);

    // Like Node::text(), but escapes the text to protect against XSS attacks
    static Node escaped(const std::string&);
};

// Subtree of a Node
typedef std::vector<Node> Children;

// Generate a new unique element ID
std::string new_id();
}
