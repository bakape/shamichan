#pragma once

#include <string>

namespace brunhild {

// Escape a user-submitted unsafe string to protect against XSS and malformed
// HTML
std::string escape(const std::string& s);
}
