#pragma once

#include "node.hh"
#include <emscripten/bind.h>
#include <functional>
#include <string>

namespace brunhild {

// Handles a captured event and receives the passed Event object as the only
// argument
typedef std::function<void(emscripten::val&)> Handler;

// Register a persistent global event handler.
// type: DOM event type (click, hover, ...).
// selector: any CSS selector the event target should be matched against
// Returns handler ID
long register_handler(
    std::string type, Handler handler, std::string selector = "");

// Remove a global event handler by ID
void unregister_handler(long id);
}
