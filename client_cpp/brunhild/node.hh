#pragma once

#include "util.hh"
#include <optional>
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>

namespace brunhild {

// Generate a new unique element ID
std::string new_id();

// Helper for serializing to HTML
class HTMLWriter {
public:
    // Renders HTML string
    std::string html();

    // Write as HTML to stream
    virtual void write_html(Rope&) = 0;
};

// Element attributes
class Attrs : public std::unordered_map<std::string, std::string>,
              public HTMLWriter {
    typedef std::unordered_map<std::string, std::string> Base;
    using Base::Base;

public:
    // Write attrs as HTML to stream
    void write_html(Rope&);

    // Diff attributes with new value and apply patches to the DOM
    void patch(Attrs&& attrs);
};

// Represents an HTML element. Can be used to construct node trees more easily.
class Node : public HTMLWriter {
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

    // Write node as HTML to stream
    void write_html(Rope&);

    // Converts the subtree of the node into an HTML string and sets it to
    // inner_html. This can reduce the diffing and memory costs of large mostly
    // static subtrees, but will cause any changes to replace the entire
    // subtree.
    void stringify_subtree();

    // Resets the node and frees up used resources
    void clear();

    // Shortcut for setting a node as hidden
    void hide();
};

// Subtree of a Node
typedef std::vector<Node> Children;
}
