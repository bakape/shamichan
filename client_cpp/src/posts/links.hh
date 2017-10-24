// Various post link rendering functions
#pragma once

#include "../../brunhild/node.hh"

// Render a link to a post with optional inlined linked post
brunhild::Node render_post_link(uint64_t id, const LinkData& data);
