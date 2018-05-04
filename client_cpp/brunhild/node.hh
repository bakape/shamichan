#pragma once

#include "util.hh"
#include <optional>
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>

namespace brunhild {

// Element attributes
class Attrs : public std::unordered_map<std::string, std::string> {
    typedef std::unordered_map<std::string, std::string> Base;
    using Base::Base;

public:
    // Write attrs as HTML to stream
    void write_html(std::ostringstream&) const;

    // Diff attributes with new value and apply patches to the DOM
    void patch(Attrs attrs);
};

// Represents an HTML element. Can be used to construct node trees more easily.
class Node {
public:
    // Tag of the Element
    std::string tag;

    // Attributes and properties of the Element
    Attrs attrs;

    // Children of the element
    std::vector<Node> children;

    // Inner HTML of the Element. If set, children are ignored
    std::optional<std::string> inner_html;

    // Creates a Node with optional attributes and children
    Node(std::string tag, Attrs attrs = {}, std::vector<Node> children = {})
        : tag(tag)
        , attrs(attrs)
        , children(children)
    {
    }

    // Creates a Node with html set as the inner contents.
    // Escaped specifies, if the text should be escaped.
    Node(std::string tag, Attrs attrs, std::string html, bool escape = false)
        : tag(tag)
        , attrs(attrs)
        , inner_html(escape ? brunhild::escape(html) : html)
    {
    }

    // Creates a Node with html set as the inner contents.
    // Escaped specifies, if the text should be escaped.
    Node(std::string tag, std::string html, bool escape = false)
        : Node(tag, {}, html, escape)
    {
    }

    Node() = default;

    // Renders Node and subtree to HTML
    std::string html() const;

    // Write node as HTML to stream
    void write_html(std::ostringstream&) const;

    // Converts the subtree of the node into an HTML string and sets it to
    // inner_html. This can reduce the diffing and memory costs of large mostly
    // static subtrees, but will cause any changes to replace the entire
    // subtree.
    void stringify_subtree();

    // Resets the node and frees up used resources
    void clear();
};

// Subtree of a Node
typedef std::vector<Node> Children;

// Renders Children to HTML with fewer allocations
std::string render_children(const Children&);

// Generate a new unique element ID
std::string new_id();
}
