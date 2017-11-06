#pragma once

#include "../../brunhild/node.hh"
#include "models.hh"
#include <ctime>
#include <string>
#include <string_view>

// Renders readable elapsed time since Unix timestamp then
std::string relative_time(time_t then);

// Render a link to a post with optional inlined linked post
brunhild::Node render_post_link(uint64_t id, const LinkData& data);

// Render and anchor link. new_tab specifies, if it opens in a new tab.
brunhild::Node render_link(
    std::string_view url, std::string_view text, bool new_tab = true);
