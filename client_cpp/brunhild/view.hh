#pragma once

#include "events.hh"
#include "node.hh"
#include <emscripten/val.h>
#include <functional>
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>

namespace brunhild {

// Base class for views implementing a virtual DOM subtree with diffing of
// passed Nodes to the current state of the DOM and appropriate pathing.
// You are not required to use this class for structureing your applications and
// can freely build your own abstractions on top of the functions in
// mutations.hh.
class View {
public:
    // Render the root node and its subtree. Chaning the "id" attribute of the
    // root node will invalidate all event_handlers for this View.
    virtual Node render() = 0;

    // Initialize the view with a Node subtree. Separate function, so you can
    // optimise DOM writes as you see fit (outside your subclass constructor)
    // and allocate View in static memory.
    // Calling more then once will overwrite previous state.
    void init();

    // Renders the view's subtree as HTML. After this call, the HTML must be
    // inserted into a parent view or passed to one of DOM mutation functions.
    std::string html() const;

    // Same as html(), but writes to a stream to reduce allocations
    void write_html(std::ostringstream&) const;

    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    void patch();

    // Removes the View from the DOM and any DOM events listeners
    virtual void remove();

    // Add DOM event handler to view. Must be called after init().
    // These will persist, until remove() is called, View is destroyed or init()
    // is called again.
    // If you have many instances of the same View subclass, it
    // is  recommended to use register_handler with View collection lookup on
    // your side to reduce DOM event listener count.
    // type: DOM event type (click, hover, ...)
    // selector: any CSS selector the event target should be matched against
    void on(std::string type, std::string selector, Handler handler);

    ~View() { remove_event_handlers(); }

    // Returns root node id. Only valid after init() has run.
    std::string id() const { return saved.attrs.at("id"); }

private:
    std::vector<long> event_handlers;

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

    void remove_event_handlers();
};
}
