#pragma once

#include "events.hh"
#include "mutations.hh"
#include "node.hh"
#include <emscripten.h>
#include <emscripten/val.h>
#include <memory>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

namespace brunhild {

// Base class for views.
// You are not required to use this class for structureing your applications and
// can freely build your own abstractions on top of the functions in
// mutations.hh.
// Note that the address to this view has to remain constant for DOM event
// handlers to work.
class View : public HTMLWriter {
public:
    // ID of root node
    const std::string id;

    // Creates new view with an optional root node ID
    View(std::string id = new_id());

    // Remove all event listeners
    virtual ~View();

    // Add DOM event handler to view.
    // If you have many instances of the same View subclass, it
    // is  recommended to use register_handler with View collection lookup on
    // your side to reduce DOM event listener count.
    // type: DOM event type (click, hover, ...)
    // selector: any CSS selector the event target should be matched against
    // handler: handler for a matched event
    void on(std::string type, std::string selector, Handler handler);

    // Removes the View from the DOM
    virtual void remove();

    // Scroll the root element of View into the viewport
    void scroll_into_view();

    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    virtual void patch() = 0;

protected:
    // Returns the root element of the view
    emscripten::val el();

private:
    // Registered DOM event handlers
    std::vector<long> event_handlers;

    void remove_event_handlers();
};

// Base class for views implementing a virtual DOM subtree with diffing of
// passed Nodes to the current state of the DOM and appropriate pacthing.
// You are not required to use this class for structureing your applications and
// can freely build your own abstractions on top of the functions in
// mutations.hh.
// Note that the address to this view has to remain constant for DOM event
// handlers to work.
class VirtualView : public View {
public:
    // Render the root node and its subtree.
    // The "id" attribute on the root node is ignored and is always set to
    // View::id.
    virtual Node render() = 0;

    // Same as html(), but writes to a stream to reduce allocations
    void write_html(Rope&);

    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    virtual void patch();

    // Creates a new View with an optional specific root node ID.
    VirtualView(std::string id = new_id())
        : View(id)
    {
    }

protected:
    // Initialize view with subtree
    virtual void init();

    // Contains data about the state of the DOM subtree after the last patch
    // call
    Node saved;

    // Ensure the Node and it's subtree all have element IDs defined
    void ensure_id(Node&);

private:
    bool is_initialized = false;

    // Patch an old node against the new one and generate DOM mutations
    void patch_node(Node& old, Node&& node);

    // Patch element's subtree
    void patch_children(Node& old, Node&& node);
};

// Simple constant view that renders a Node with its subtree
class NodeView : public VirtualView {
public:
    NodeView(Node n) { saved = n; }
    Node render() { return saved; }
    void patch() {}

protected:
    void init()
    {
        saved.attrs["id"] = id;
        ensure_id(saved);
    }
};

// Utility adapter for the MV* pattern
template <class M> class ModelView : public VirtualView {
public:
    // Caches model pointer and calls render_model(M*)
    Node render() final
    {
        m = get_model();
        if (!m) {
            EM_ASM_INT(
                {
                    console.error('model missing on view: ' + UTF8ToString($0));
                },
                id.data());
            throw "model missing";
        }
        return render(m);
    }

    // Fetches pointer to model (for example, from some collection or weak
    // pointer). Must return NULL, if model no longer exists.
    virtual M* get_model() = 0;

protected:
    // Cached pointer to model. Set right before calling render(M*).
    M* m;

    // Render the root node and its subtree, according to model.
    // The "id" attribute on the root node is ignored and is always set to
    // View::id.
    virtual Node render(M*) = 0;
};

// Common functionality of all parent views
template <class V = View> class ParentView : public View {
public:
    // Tag of root node
    const std::string tag;

    // Creates a new view with an optional specific root node ID.
    ParentView(std::string tag, std::string id = new_id())
        : View(id)
        , tag(tag)
    {
    }

    // Same as html(), but writes to a stream to reduce allocations
    void write_html(Rope& s)
    {
        if (!is_initialized) {
            init();
            is_initialized = true;
        }

        s << '<' << tag;
        saved_attrs.write_html(s);
        s << '>';
        for (auto v : saved) {
            v->write_html(s);
        }
        s << "</" << tag << '>';
    }

protected:
    // Last rendered attributes
    Attrs saved_attrs;

    // List of views saved since last diff by ID
    std::vector<std::shared_ptr<V>> saved;

    // Returns the attributes of the container view
    virtual Attrs attrs() { return {}; };

    virtual void init()
    {
        saved_attrs = attrs();
        saved_attrs["id"] = id;
    }

private:
    bool is_initialized = false;
};

// Renders and manages a list of views using a delegator method.
// M: model
// V: ModelView<M>
template <class M, class V> class ListView : public ParentView<V> {
    using ParentView<V>::ParentView;
    using ParentView<V>::saved;
    using ParentView<V>::saved_attrs;
    using ParentView<V>::attrs;

public:
    virtual void init()
    {
        for (auto m : get_list()) {
            saved.push_back(create_child(m));
        }
        ParentView<V>::init();
    }

    // Patches the attributes of the ListView and reorders its children, while
    // also creating any missing ones an removing no longer actual children.
    // deep: should patching recurse to the view's child views
    void patch()
    {
        saved_attrs.patch(attrs());

        const auto new_list = get_list();
        const auto new_set
            = std::unordered_set<M*>(new_list.begin(), new_list.end());
        std::unordered_map<M*, std::shared_ptr<V>> saved_set;
        std::vector<M*> saved_list;

        // Map saved views to models
        saved_set.reserve(saved.size());
        saved_list.reserve(saved.size());
        for (auto it = saved.begin(); it != saved.end();) {
            auto& v = *it;
            auto m = v->get_model();
            if (m) {
                saved_set[m] = v;
                saved_list.push_back(m);
                it++;
            } else {
                // Get rid of views without models
                v->remove();
                it = saved.erase(it);
            }
        }

        // Diff and reorder views in the overlaping range
        for (size_t i = 0; i < saved_list.size() && i < new_list.size(); i++) {
            auto m = new_list[i];
            if (saved_list[i] == m) {
                saved_set[m]->patch();
                saved_set.erase(m);
                continue;
            }

            std::shared_ptr<V> v;
            if (saved_set.count(m)) {
                v = saved_set.at(m);
                if (!i) {
                    move_prepend(View::id, v->id);
                } else {
                    move_after(saved[i - 1]->id, v->id);
                }
                saved_set[m]->patch();
                saved_set.erase(m);
            } else {
                v = create_child(m);
                if (!i) {
                    prepend(View::id, v->html());
                } else {
                    after(View::id, v->html());
                }
            }
            saved[i] = v;
        }

        if (saved.size() > new_list.size()) {
            // Remove all unused old views
            for (auto& p : saved_set) {
                p.second->remove();
            }
            saved.resize(new_list.size());
        } else {
            // Append all missing views
            for (size_t i = saved.size() - 1; i < new_list.size(); i++) {
                append(View::id,
                    saved.emplace_back(create_child(new_list[i]))->html());
            }
        }
    }

protected:
    // Returns an ordered list of models to be used to render view contents
    virtual std::vector<M*> get_list() = 0;

    // Create a new instance of a child view
    virtual std::shared_ptr<V> create_child(M*) = 0;
};

// Combines multiple views as its children. The list and order of the child
// views never mutates.
template <class V = View> class CompositeView : public ParentView<V> {
    using ParentView<V>::ParentView;
    using ParentView<V>::saved;
    using ParentView<V>::saved_attrs;
    using ParentView<V>::attrs;

public:
    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    // deep: should patching recurse to the view's child views
    void patch()
    {
        saved_attrs.patch(attrs());
        for (auto& v : saved) {
            v->patch();
        }
    }

protected:
    virtual void init()
    {
        const auto list = get_list();
        saved.reserve(list.size());
        for (auto v : list) {
            saved.emplace_back(v);
        }
        ParentView<V>::init();
    }

    // Returns an ordered list of views to be used as children of this view.
    // This method is only called once.
    virtual std::vector<V*> get_list() = 0;
};
}
