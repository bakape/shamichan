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

// Base class for all views
class BaseView {
public:
    // ID of root node
    const std::string id;

    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    // deep: should patching recurse to the view's child views
    virtual void patch(bool deep = false) = 0;

    // Renders the view's subtree as HTML
    std::string html();

    // Same as html(), but writes to a stream to reduce allocations
    virtual void write_html(Rope&) = 0;

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
    // handler: handler for a matched event
    void on(std::string type, std::string selector, Handler handler);

    // Get pointer to a view by ID or NULL, if none
    template <class T> static T* get(const std::string& id)
    {
        if (!BaseView::instances.count(id)) {
            return 0;
        }
        return dynamic_cast<T*>(BaseView::instances.at(id).get());
    }

    // Initializes view as a root node of a view tree. Returns HTML to be
    // inserted into the DOM.
    std::string init_as_root();

    // Removes the View from the DOM. Can only be called from the parent owning
    // the view. The view is destructed after this and can no longer be used.
    virtual void remove();

protected:
    // Register view in global collection and returns a pointer to it.
    // It is paramount that no view IDs ever
    // collide.
    static void store(BaseView* v);

private:
    // Registered DOM event handlers
    std::vector<long> event_handlers;

    void remove_event_handlers();

    // All existing view instances
    static std::unordered_map<std::string, std::unique_ptr<BaseView>> instances;
};

// Base class for views implementing a virtual DOM subtree with diffing of
// passed Nodes to the current state of the DOM and appropriate pathing.
// You are not required to use this class for structureing your applications and
// can freely build your own abstractions on top of the functions in
// mutations.hh.
class View : public BaseView {
public:
    // Render the root node and its subtree.
    // The "id" attribute on the root node is ignored and is always set to
    // BaseView::id.
    virtual Node render() = 0;

    // Same as html(), but writes to a stream to reduce allocations
    void write_html(Rope&);

    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    // deep: does nothing on this class
    virtual void patch(bool deep = false);

    // Creates a new View with an optional specific root node ID.
    View(std::string id = new_id())
        : BaseView(id)
    {
    }

protected:
    // Initialize view with subtree
    virtual void init();

private:
    bool is_initialized = false;

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
class Model {
public:
    // Returns, if the model's view needs to be diffed and pathced. Can be used
    // to optimize patching frequency or create constant views.
    virtual bool need_patch() { return true; }
};

// Utility adapter for the MV* pattern
template <class M> class ModelView : public View {
public:
    // Caches model pointer and calls render_model()
    Node render()
    {
        m = get_model();
        return render_model();
    }

    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    // deep: does nothing on this class
    void patch(bool deep = false)
    {
        if (get_model()->need_patch()) {
            View::patch(deep);
        }
    }

    // Fetches pointer to model (for example, from some collection or weak
    // pointer). Must return NULL, if model no longer exists.
    virtual M* get_model() = 0;

protected:
    // Cached pointer to model. Set right before calling render_model().
    M* m;

    // Render the root node and its subtree, according to model.
    // The "id" attribute on the root node is ignored and is always set to
    // BaseView::id.
    virtual Node render_model() = 0;
};

// Common functionality of all parent views
template <class V> class ParentView : public BaseView {
public:
    // Tag of root node
    const std::string tag;

    // Creates a new view with an optional specific root node ID.
    ParentView(std::string tag, std::string id = new_id())
        : BaseView(id)
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
    std::vector<V*> saved;

    // Returns the attributes of the container view
    virtual Attrs attrs() const { return {}; };

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
    void patch(bool deep = false)
    {
        saved_attrs.patch(attrs());

        const auto new_list = get_list();
        const auto new_set
            = std::unordered_set<M*>(new_list.begin(), new_list.end());
        std::unordered_map<M*, V*> saved_set;
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

        if (deep) {
            for (auto& p : saved_set) {
                if (new_set.count(p.first)) {
                    p.second->patch(deep);
                }
            }
        }

        // Diff and reorder views in the overlaping range
        for (size_t i = 0; i < saved_list.size() && i < new_list.size(); i++) {
            auto m = new_list[i];
            if (saved_list[i] == m) {
                saved_set.erase(m);
                continue;
            }

            V* v;
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
                BaseView::store(v);
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
                auto v = create_child(new_list[i]);
                append(BaseView::id, v->html());
                saved.push_back(v);
            }
        }
    }

protected:
    // Returns an ordered list of models to be used to render view contents
    virtual std::vector<M*> get_list() = 0;

    // Create a new instance of a child view
    virtual V* create_child(M*) = 0;
};

// Combines multiple views as its children. The list and order of the child
// views never mutates.
template <class V = BaseView> class CompositeView : public ParentView<V> {
    using ParentView<V>::ParentView;
    using ParentView<V>::saved;
    using ParentView<V>::saved_attrs;
    using ParentView<V>::attrs;

public:
    // Patch the view's subtree against the updated subtree.
    // Can only be called after the view has been inserted into the DOM.
    // deep: should patching recurse to the view's child views
    void patch(bool deep = false)
    {
        saved_attrs.patch(attrs());
        if (deep) {
            for (auto& v : saved) {
                v->patch(deep);
            }
        }
    }

protected:
    virtual void init()
    {
        saved = get_list();
        for (auto v : saved) {
            BaseView::store(v);
        }
        ParentView<V>::init();
    }

    // Returns an ordered list of views to be used as children of this view.
    // This method is only called once.
    virtual std::vector<V*> get_list() = 0;
};
}
