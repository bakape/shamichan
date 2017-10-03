#pragma once

#include <string>

void local_storage_set(const std::string key, const std::string val);
std::string local_storage_get(const std::string key);
