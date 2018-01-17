#pragma once

// Pixels to scroll the page by on next RAF
extern int scroll_by;

// Scroll page by an amount to compensate for DOM mutations or top banner height
// as needed
void compensate_scrolling();

// Init listeners and values for scrolling compensation
void init_scrolling();

// Scroll to a post and compensate for the banner height
void scroll_to_post(unsigned id);
