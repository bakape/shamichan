#pragma once

#include "node.hh"
#include <optional>
#include <sstream>
#include <string>
#include <vector>

namespace brunhild {

// Base class for structured DOM view representations.
// This is for convience only and you are not required to use this class to
// structure your application.
class View {
public:
    // ID of the element
    const std::string id;

    // Constructs a View with an optional element ID. If none is specified, a
    // unique ID is automatically generated.
    View(std::string id = new_id())
        : id(id)
    {
    }

    // Append a node as an HTML string to the view's DOM element
    void append(std::string html);

    // Prepend a node as an HTML string to the view's DOM element
    void prepend(std::string html);

    // Insert a node as an HTML string before this view's DOM element
    void before(std::string html);

    // Insert a node as an HTML string after this view's DOM element
    void after(std::string html);

    // Sets the inner HTML of the view's DOM element
    // More efficient than individual appends, etc.
    void set_inner_html(std::string html);

    // Sets the children of the view's DOM element. This removes the previous
    // children of the element.
    // More efficient than individual appends, etc.
    void set_children(const Children&);

    // Remove the view's element from the DOM. The view should be considered in
    // an invalid state after this.
    void remove();

    // Set the value of an attribute on the view's DOM element
    void set_attr(std::string key, std::string val);

    // Remove an attribute from the view's DOM element
    void remove_attr(std::string key);
};

// Base class for views implementing a virtual DOM subtree with diffing of
// passed Nodes to the current state of the DOM and appropriate pathing.
class VirtualView {
public:
    // Initialize the view with a Node subtree
    void init(Node);

    // Renders the view's subtree as HTML. After this call, the HTML must be
    // inserted into a parent view or passed to one of DOM mutation functions.
    std::string html() const;

    // Same as html(), but writes to a stream to reduce allocations
    void write_html(std::ostringstream&) const;

    // Patch the view's subtree against the updated subtree in Node.
    // Can only be called after the view has been inserted into the DOM.
    void patch(Node);

private:
    // Contains data about the state of the DOM subtree after the last patch
    // call
    Node saved;

    // Ensure the Node and it's subtree all have element IDs defined
    void ensure_id(Node&);

    // Patch an old node against the new one and generate DOM mutations
    void patch_node(Node& old, Node node);

    // Patch element attributes
    void patch_attrs(Node& old, Attrs attrs);

    // Patch element's subtree
    void patch_children(Node& old, Children children);
};
}
