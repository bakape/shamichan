#pragma once

#include "../../brunhild/node.hh"
#include "models.hh"
#include <ctime>
#include <string>

// Renders readable elapsed time since Unix timestamp then
std::string relative_time(time_t then);

// Render a link to a post with optional inlined linked post
brunhild::Node render_post_link(uint64_t id, const LinkData& data);
