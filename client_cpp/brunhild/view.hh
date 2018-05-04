#pragma once

#include "events.hh"
#include "mutations.hh"
#include "node.hh"
#include <emscripten.h>
#include <emscripten/val.h>
#include <functional>
#include <memory>
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>

namespace brunhild {

// Base class for all views.
// Note: for DOM event handlers to work the memory address of the view must not
// change. This can be done by allocating it to the heap through std::shared_ptr
// or std::unique_ptr.
class BaseView {
public:
    // ID of root node
    const std::string id;

    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    virtual void patch() = 0;

    // Renders the view's subtree as HTML
    virtual std::string html() const = 0;

    // Same as html(), but writes to a stream to reduce allocations
    virtual void write_html(Rope&) const = 0;

    // Creates new view with an optional root node ID
    BaseView(std::string id = new_id())
        : id(id)
    {
    }

    virtual ~BaseView() { remove_event_handlers(); }

    // Add DOM event handler to view.
    // If you have many instances of the same View subclass, it
    // is  recommended to use register_handler with View collection lookup on
    // your side to reduce DOM event listener count.
    // type: DOM event type (click, hover, ...)
    // selector: any CSS selector the event target should be matched against
    void on(std::string type, std::string selector, Handler handler);

private:
    // Registered DOM event handlers
    std::vector<long> event_handlers;

    void remove_event_handlers();
};

// Base class for views implementing a virtual DOM subtree with diffing of
// passed Nodes to the current state of the DOM and appropriate pathing.
// You are not required to use this class for structureing your applications and
// can freely build your own abstractions on top of the functions in
// mutations.hh.
// NB: init() must be called in the constructor of the top-most class that
// overrides render().
class View : public BaseView {
public:
    // Render the root node and its subtree.
    // The "id" attribute on the root node is ignored and is always set to
    // BaseView::id.
    virtual Node render() = 0;

    // Removes the View from the DOM
    virtual void remove();

    // Renders the view's subtree as HTML
    std::string html() const;

    // Same as html(), but writes to a stream to reduce allocations
    void write_html(Rope&) const;

    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    void patch();

    // Initialize view with subtree. Must be called in the constructor of the
    // topmost class that overrides render().
    void init();

    // Creates a new View with an optional specific root node ID.
    View(std::string id = new_id())
        : BaseView(id)
    {
    }

private:
    // Contains data about the state of the DOM subtree after the last patch
    // call
    Node saved;

    // Ensure the Node and it's subtree all have element IDs defined
    void ensure_id(Node&);

    // Patch an old node against the new one and generate DOM mutations
    void patch_node(Node& old, Node node);

    // Patch element's subtree
    void patch_children(Node& old, Node node);
};

// Utility adapter for the MV* pattern
// NB: View::init() must be called in the constructor of the top-most class that
// overrides render_model().
template <class M> class ModelView : public View {
public:
    // Caches model pointer and calls render_model()
    Node render()
    {
        m = get_model();
        return render_model();
    }

    // Fetches pointer to model (for example, from some collection).
    // Must return NULL, if model no longer exists.
    virtual M* get_model() = 0;

protected:
    // Cached pointer to model. Set right before calling render_model().
    M* m;

    // Render the root node and its subtree, according to model.
    // The "id" attribute on the root node is ignored and is always set to
    // BaseView::id.
    virtual Node render_model() = 0;
};

// Renders and manages a list of views using a delegator method.
// M: model
// V: ModelView<M>
template <class M, class V> class ListView : public BaseView {
public:
    // Tag of root node
    const std::string tag;

    // Creates a new view with an optional specific root node ID.
    ListView(std::string tag, std::string id = new_id())
        : BaseView(id)
        , tag(tag)
    {
    }

    // The last deriving class to override method must call init() in its
    // constructor to  initialize the ListView
    void init()
    {
        for (auto m : get_list()) {
            saved.push_back(create_child(m));
        }
        saved_attrs = attrs();
        saved_attrs["id"] = id;
    }

    // Patches the attributes of the ListView and reorders its children, while
    // also creating any missing ones an removing no longer actual children.
    // Note: For performance reasons this method does not patch the entire
    // subtree. To do a deep patch call patch_deep().
    void patch()
    {
        saved_attrs.patch(attrs());

        const auto new_list = get_list();
        std::unordered_map<M*, std::shared_ptr<V>> saved_set;
        std::vector<M*> saved_list;

        // Map saved views to models
        saved_set.reserve(saved.size());
        saved_list.reserve(saved.size());
        for (auto it = saved.begin(); it != saved.end();) {
            auto v = *it;
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
                saved_set.erase(m);
                continue;
            }

            std::shared_ptr<V> v;
            if (saved_set.count(m)) {
                v = saved_set.at(m);
                if (!i) {
                    move_prepend(BaseView::id, v->id);
                } else {
                    move_after(saved[i - 1]->id, v->id);
                }
                saved_set.erase(m);
            } else {
                v = create_child(m);
                if (!i) {
                    prepend(BaseView::id, v->html());
                } else {
                    after(BaseView::id, v->html());
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
                saved.push_back(create_child(new_list[i]));
            }
        }
    }

    // Patches the view and it's entire subtree
    void patch_deep()
    {
        patch();
        for (auto v : saved) {
            v->patch();
        }
    }

    // Renders the view's subtree as HTML
    std::string html() const
    {
        Rope s;
        write_html(s);
        return s.str();
    }

    // Same as html(), but writes to a stream to reduce allocations
    void write_html(Rope& s) const
    {
        s << '<' << tag;
        saved_attrs.write_html(s);
        s << '>';
        for (auto& v : saved) {
            v->write_html(s);
        }
        s << "</" << tag << '>';
    }

protected:
    // Returns the attributes of the container view
    virtual Attrs attrs() const { return {}; };

    // Returns an ordered list of models to be used to render view contents
    virtual std::vector<M*> get_list() = 0;

    // Create a new instance of a child view
    virtual std::shared_ptr<V> create_child(M*) = 0;

private:
    // List of views since last render
    std::vector<std::shared_ptr<V>> saved;

    // Last rendered attributes
    Attrs saved_attrs;
};
}
