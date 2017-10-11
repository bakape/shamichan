#include "mutations.hh"
#include <emscripten.h>
#include <set>

namespace brunhild {
using std::string;

// All pending mutations quickly accessible by element ID
std::unordered_map<string, Mutations> mutations;

// Stores mutation order, so we can somewhat make sure, new children are not
// manipulated, before insertion.
std::set<string> mutation_order;

// Push simple vector-based mutation to stack
#define push_mutation(typ)                                                     \
    auto& mut = mutations[id];                                                 \
    mut.typ.push_back(html);                                                   \
    mutation_order.insert(id);

void append(string id, string html) { push_mutation(append) }

void prepend(string id, string html) { push_mutation(prepend) }

void before(string id, string html) { push_mutation(before) }

void after(string id, string html) { push_mutation(after) }

void set_inner_html(string id, string html)
{
    auto& mut = mutations[id];

    // These would be overwritten, so we can free up used memory
    mut.free_inner();

    mut.set_inner_html = html;
    mutation_order.insert(id);
}

void set_outer_html(string id, string html)
{
    auto& mut = mutations[id];
    mut.free_outer();
    mut.set_outer_html = html;
    mutation_order.insert(id);
}

void remove(string id)
{
    auto& mut = mutations[id];
    mut.free_outer();
    mut.remove_el = true;
    mutation_order.insert(id);
}

void set_attr(string id, string key, string val)
{
    auto& mut = mutations[id];
    mut.set_attr[key] = val;
    mutation_order.insert(id);
}

void remove_attr(string id, string key)
{
    auto& mut = mutations[id];
    mut.set_attr.erase(key);
    mutation_order.insert(key);
}

void Mutations::free_inner()
{
    append.clear();
    prepend.clear();
    set_inner_html.clear();
}

void Mutations::free_outer()
{
    free_inner();
    remove_attr.clear();
    set_attr.clear();
    set_outer_html.clear();
}

extern "C" void flush()
{
    for (const string& id : mutation_order) {
        mutations.at(id).exec(id);
    }
    mutation_order.clear();
    mutations.clear();
}

void Mutations::exec(const string& id)
{
    // Assign element to global variable, so we don't have to look it up each
    // time
    const bool exists = (bool)EM_ASM_INT(
        {
            window.__el = document.getElementById(Pointer_stringify($0));
            return !!window.__el;
        },
        id.c_str());
    if (!exists) {
        // Nothing we can do
        return;
    }

    // TODO: Do these loops in one JS call, if possible

    // Before and after inserts need to happen, even if the element is going to
    // be removed
    for (auto& html : before) {
        EM_ASM_INT(
            {
                var el = window.__el;
                var cont = document.createElement('div');
                cont.innerHTML = Pointer_stringify($0);
                el.parentNode.insertBefore(cont.firstChild, el);
            },
            html.c_str());
    }
    for (auto& html : after) {
        EM_ASM_INT(
            {
                var el = window.__el;
                var cont = document.createElement('div');
                cont.innerHTML = Pointer_stringify($0);
                el.parentNode.insertBefore(cont.firstChild, el.nextSibling);
            },
            html.c_str());
    }

    if (remove_el) {
        EM_ASM({
            var el = window.__el;
            el.parentNode.removeChild(el);
        });
        // If the element is to be removed, nothing else needs to be done
        return;
    }

    if (set_outer_html.size()) {
        EM_ASM_INT({ window.__el.outerHTML = Pointer_stringify($0); },
            set_outer_html.c_str());
    }
    if (set_inner_html.size()) {
        EM_ASM_INT({ window.__el.innerHTML = Pointer_stringify($0); },
            set_inner_html.c_str());
    }

    for (auto& html : append) {
        EM_ASM_INT(
            {
                var el = window.__el;
                var cont = document.createElement('div');
                cont.innerHTML = Pointer_stringify($0);
                el.parentNode.insertBefore(cont.firstChild, el.nextSibling);
            },
            html.c_str());
    }
    for (auto& html : prepend) {
        EM_ASM_INT(
            {
                var el = window.__el;
                var cont = document.createElement('div');
                cont.innerHTML = Pointer_stringify($0);
                el.insertBefore(cont.firstChild, el.firstChild);
            },
            html.c_str());
    }

    for (auto& kv : set_attr) {
        EM_ASM_INT(
            {
                window.__el.setAttribute(
                    Pointer_stringify($0), Pointer_stringify($1));
            },
            kv.first.c_str(), kv.second.c_str());
    }
    for (auto& key : remove_attr) {
        EM_ASM_INT({ window.__el.removeAttribute(Pointer_stringify($0)); },
            key.c_str());
    }
}
}
