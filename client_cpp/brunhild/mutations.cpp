#include "mutations.hpp"
#include <set>
#include <unordered_map>

// All pending mutations quickly accessible by element ID
std::unordered_map<string, Mutations> mutations;

// Stores mutation order, so we can somewhat make sure, new children are not
// manipulated, before insertion.
std::set<string> mutation_order;

// Push simple vector-based mutation to stack
#define push_mutation(typ)                                                     \
    Mutations& mut = mutations[id];                                            \
    mut.typ.push_back(html);                                                   \
    mutation_order.insert(id);

void append(string id, string html) { push_mutation(append) }

void prepend(string id, string html) { push_mutation(prepend) }

void before(string id, string html) { push_mutation(before) }

void after(string id, string html) { push_mutation(after) }

void set_inner_html(string id, string html)
{
    Mutations& mut = mutations[id];

    // These would be overwritten, so we can free up used memory
    mut.free_inner();

    mut.set_inner_html = html;
    mutation_order.insert(id);
}

void set_outer_html(string id, string html)
{
    Mutations& mut = mutations[id];
    mut.free_outer();
    mut.set_outer_html = html;
    mutation_order.insert(id);
}

void remove(string id)
{
    Mutations& mut = mutations[id];
    mut.free_outer();
    mut.remove = true;
    mutation_order.insert(id);
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
    set_outer_html.clear();
    set_attr.clear();
}
