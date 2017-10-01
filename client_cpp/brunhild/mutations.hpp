#pragma once

#include <string>
#include <tuple>
#include <vector>

using std::string;
using std::vector;
using std::tuple;

// Pending mutations for an element
class Mutations {
public:
    bool remove;
    string set_inner_html, set_outer_html;
    vector<string> append, prepend, before, after, remove_attr;
    vector<tuple<string, string>> set_attr;

    // Clear mutations of element inner content to free up memory
    void free_inner();

    // Clear mutations of element inner and outer content to free up memory
    void free_outer();
};

// Append a node to a parent
void append(string id, string html);

// Prepend a node to a parent
void prepend(string id, string html);

// Insert a node before a sibling
void before(string id, string html);

// Insert a node after a sibling
void after(string id, string html);

// Set inner html of an element
void set_inner_html(string id, string html);

// Set outer html of an element
void set_inner_html(string id, string html);

// Remove an element
void remove(string id);
