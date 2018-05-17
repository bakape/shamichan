#pragma once

#include <functional>
#include <string>

namespace brunhild {
// Append a node to a parent
void append(std::string id, std::string html);

// Prepend a node to a parent
void prepend(std::string id, std::string html);

// Move child node to the front of the parent
void move_prepend(std::string parent_id, std::string child_id);

// Move child node after a sibling in the parent
void move_after(std::string sibling_id, std::string child_id);

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

// Scroll and element into the viewport
void scroll_into_view(std::string id);

// Flush all pending DOM mutations
extern "C" void flush();

// Function to run before flushing DOM updates. Is run on each call of flush().
extern void (*before_flush)();

// Function to run after flushing DOM updates. IIs run on each call of flush().
extern void (*after_flush)();
}
