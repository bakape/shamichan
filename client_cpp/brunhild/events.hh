#pragma once

#include "node.hh"
#include <emscripten/bind.h>
#include <string>

namespace brunhild {

// Handles a captured event and receives the passed Event object as the only
// argument
typedef void (*Handler)(emscripten::val&);

// Register a persistent global event handler.
// type specifies DOM event type (click, hover, ...).
// selector specifies any CSS selector the event target should be matched
// against
// Returns handler ID.
long register_handler(
    std::string type, Handler handler, std::string selector = "");

// Remove a global event handler by ID
void unregister_handler(long id);
}
