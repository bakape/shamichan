#pragma once

#include "events.hh"
#include "node.hh"
#include <optional>
#include <sstream>
#include <string>
#include <unordered_set>
#include <vector>

namespace brunhild {

// Base class for structured DOM view representations.
// This is for convience only and you are not required to use this class to
// structure your application.
class View {
protected:
    // Describes an event to be prevented or handled
    struct EventFilter {
        std::string type, // DOM event type (click, hover, ...)
            selector; // specifies any CSS selector the event target should be
                      // matched against
    };

    // Handlers for events on the root node or inside view's subtree.
    // Override this to handle DOM events.
    const std::vector<std::pair<EventFilter, Handler>> event_handlers;

public:
    // ID of the element
    const std::string id;

    // Method of attaching the View's root element to its parent
    enum class InsertionMode { append, prepend, before, after };

    // Constructs a View with an optional and attaches it to parent element.
    // node is the root node and subtree of the element.
    // If root element ID is not specified, a unique ID is automatically
    // generated. mode sets in what way the element is attached to its parent.
    View(const std::string& parent_id, Node node,
        InsertionMode mode = InsertionMode::append);

    // Unregisters any event handlers
    ~View()
    {
        for (auto id : event_handler_ids) {
            unregister_handler(id);
        }
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

private:
    std::vector<long> event_handler_ids;
};

// Base class for views implementing a virtual DOM subtree with diffing of
// passed Nodes to the current state of the DOM and appropriate pathing.
class VirtualView {
public:
    // Initialize the view with a Node subtree. Takes ownership of Node.
    // Calling more then once will overwrite previous state.
    void init(Node);

    // Renders the view's subtree as HTML. After this call, the HTML must be
    // inserted into a parent view or passed to one of DOM mutation functions.
    std::string html() const;

    // Same as html(), but writes to a stream to reduce allocations
    void write_html(std::ostringstream&) const;

    // Patch the view's subtree against the updated subtree in Node.
    // Can only be called after the view has been inserted into the DOM.
    // Takes ownership of Node.
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
    void patch_children(Node& old, Node node);
};
}
