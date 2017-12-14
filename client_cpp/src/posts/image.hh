#pragma once

#include "../../brunhild/events.hh"

// Image click handler
void handle_image_click(const brunhild::EventTarget&);

// Reveal/hide thumbnail by clicking [Show]/[Hide] in hidden thumbnail mode
void toggle_hidden_thumbnail(const brunhild::EventTarget&);
