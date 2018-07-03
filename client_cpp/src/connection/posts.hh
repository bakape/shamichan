// Post-related websocket message handlers

#pragma once

#include <string_view>

// Insert a post into the thread from JSON
void insert_post(std::string_view);
