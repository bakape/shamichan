#pragma once

#include "../../brunhild/node.hh"
#include <string>

// Render the page, after all the internal state has been loaded
void render_page();

// Format and escape a page title
std::string format_title(const std::string& board, const std::string& text);

// Set's the title of the page. Requires escaping of the string.
void set_title(std::string);

// Renders a clickable button element.
// If href = std::nullopt, no href property is set on the link.
// If aside = true, renders the button as an <aside> element, instead of <span>.
brunhild::Node render_button(
    std::optional<std::string> href, std::string text, bool aside = false);

// Push board-specific hover-revealed information elements, if any, to ch.
void push_board_hover_info(brunhild::Children& ch);
