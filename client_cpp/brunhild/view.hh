#pragma once

#include "node.hh"
#include <optional>
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

    // Append a Node to the view's DOM element
    void append(const Node&);

    // Prepend a node as an HTML string to the view's DOM element
    void prepend(std::string html);

    // Prepend a Node to the view's DOM element
    void prepend(const Node&);

    // Insert a node as an HTML string before this view's DOM element
    void before(std::string html);

    // Insert a Node before this view's DOM element
    void before(const Node&);

    // Insert a node as an HTML string after this view's DOM element
    void after(std::string html);

    // Insert a Node after this view's DOM element
    void after(const Node&);

    // Sets the inner HTML of the view's DOM element
    // More efficient than individual appends, etc.
    void set_inner_html(std::string html);

    // Sets the children of the view's DOM element. This removes the previous
    // children of the element.
    // More efficient than individual appends, etc.
    void set_children(const std::vector<Node>&);

    // Remove the view's element from the DOM. The view should be considered in
    // an invalid state after this.
    void remove();

    // Set the value of an attribute on the view's DOM element
    void set_attr(std::string key, std::string val);

    // Remove an attribute from the view's DOM element
    void remove_attr(std::string key);
};
}
