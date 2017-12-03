#pragma once

#include "../brunhild/node.hh"
#include <cctype>
#include <ostream>
#include <string>
#include <string_view>
#include <tuple>

// A string_view constructable from an owned char*, that also takes ownership
// of char* and frees it on drop.
class c_string_view : public std::string_view {
public:
    // Takes ownership of char*. char* must not be NULL.
    c_string_view(char* s)
        : std::string_view(s)
        , ch(s)
    {
    }

    ~c_string_view() { delete[] ch; }

private:
    char* ch;
};

// Read inner HTML from DOM element by ID
c_string_view get_inner_html(const std::string& id);

// Return either the singular or plural form of a translation, depending on n
std::string pluralize(int n, const std::tuple<std::string, std::string>& word);

// URL encode a string to pass into an ostream
struct url_encode {
    url_encode(const std::string& s)
        : str(s)
    {
    }

    friend std::ostream& operator<<(std::ostream& os, const url_encode& u);

private:
    const std::string& str;

    constexpr static char hex[17] = "0123456789abcdef";

    // Converts character code to hex to HEX
    static char to_hex(char code) { return hex[code & 15]; }
};

// Render submit button with and optional cancel button
brunhild::Children render_submit(bool cancel);

namespace console {
// Log string to JS console
void log(const std::string&);

// Log string to JS console as warning
void warn(const std::string&);

// Log string to JS console as error
void error(const std::string&);
}
