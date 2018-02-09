#pragma once

#include "../../brunhild/node.hh"
#include "models.hh"
#include <ctime>
#include <string>
#include <string_view>

// Checkbox for toggling deleted post display
const brunhild::Node delete_toggle = brunhild::Node("input",
    {
        { "type", "checkbox" },
        { "class", "deleted-toggle" },
    });

// Renders readable elapsed time since Unix timestamp then
std::string relative_time(time_t then);

// Generate absolute URL of a thread
std::string absolute_thread_url(unsigned long id, std::string board);

// Render a link to a post with optional inlined linked post.
brunhild::Node render_post_link(unsigned long id, const LinkData& data);

// Render and anchor link. new_tab specifies, if it opens in a new tab.
brunhild::Node render_link(
    std::string_view url, std::string_view text, bool new_tab = true);

// Match target post by attributes of an element. If none found, returns NULL.
Post* match_post(const brunhild::Attrs&);
