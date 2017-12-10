#pragma once

#include "../../brunhild/node.hh"
#include <optional>
#include <string_view>

// Parse word for possible URL handling. Returns link or embed Node, if matched.
// The validation is very trivial and mainly just checks for inline JS.
std::optional<brunhild::Node> parse_url(std::string_view word);
