#pragma once

#include <optional>
#include <string>

// Set a localStorage value
void local_storage_set(const std::string& key, const std::string& val);

// Set a localStorage value by key. If key does not exist, returns an empty
// string.
std::optional<std::string> local_storage_get(const std::string& key);
