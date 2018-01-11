#pragma once

#include "node.hh"
#include <string>

namespace brunhild {

// Describes the target node of the event
struct EventTarget {
    // Tag of the element
    std::string tag;

    // Attributes of the element
    Attrs attrs;
};

// Handles a captured event
typedef void (*Handler)(const EventTarget&);

// Register a persistent global event handler.
// type specifies DOM event type (click, hover, ...).
// selector specifies any CSS selector the event target should be matched
// against
void register_handler(
    std::string type, Handler handler, std::string selector = "");
}
