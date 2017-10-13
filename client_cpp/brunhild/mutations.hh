#pragma once

#include <string>
#include <tuple>
#include <unordered_map>
#include <vector>

namespace brunhild {
// Pending mutations for an element
class Mutations {
public:
    bool remove_el;
    std::string set_inner_html, set_outer_html;
    std::vector<std::string> append, prepend, before, after, remove_attr;
    std::unordered_map<std::string, std::string> set_attr;

    // Clear mutations of element inner content to free up memory
    void free_inner();

    // Clear mutations of element inner and outer content to free up memory
    void free_outer();

    // Execute buffered mutations
    void exec(const std::string& id);
};

// Append a node to a parent
void append(std::string id, std::string html);

// Prepend a node to a parent
void prepend(std::string id, std::string html);

// Insert a node before a sibling
void before(std::string id, std::string html);

// Insert a node after a sibling
void after(std::string id, std::string html);

// Set inner html of an element
void set_inner_html(std::string id, std::string html);

// Set outer html of an element
void set_outer_html(std::string id, std::string html);

// Remove an element
void remove(std::string id);

// Set an element attribute to a value
void set_attr(std::string id, std::string key, std::string val);

// Remove an element attribute
void remove_attr(std::string id, std::string key);

// Flush all pending DOM mutations
extern "C" void flush();
}
