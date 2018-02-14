#pragma once

#include "../../brunhild/events.hh"

// Image click handler
void handle_image_click(emscripten::val&);

// Reveal/hide thumbnail by clicking [Show]/[Hide] in hidden thumbnail mode
void toggle_hidden_thumbnail(emscripten::val&);
