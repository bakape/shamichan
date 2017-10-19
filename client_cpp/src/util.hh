#pragma once

#include <cctype>
#include <ostream>
#include <string>
#include <tuple>

// Read inner HTML from DOM element by ID
std::string get_inner_html(const std::string& id);

// Return either the singular or plural form of a translation, depending on n
std::string pluralize(int n, const std::tuple<std::string, std::string>& word);

// Cast a C string represented as int to std::string and free the original
std::string convert_c_string(int);

// URL encode a string to pass into ostream
struct url_encode {
    url_encode(const std::string& s)
        : str(s)
    {
    }

    friend std::ostream& operator<<(std::ostream& os, const url_encode& u)
    {
        for (auto ch : u.str) {
            // Keep alphanumeric and other accepted characters intact
            if (isalnum(ch)) {
                os << ch;
                continue;
            }
            switch (ch) {
            case '-':
            case '_':
            case '.':
            case '~':
                os << ch;
                break;
            case ' ':
                os << '+';
                break;
            default:
                // Any other characters are percent-encoded.
                os << '%' << to_hex(ch >> 4) << to_hex(ch & 15);
            }
        }
        return os;
    }

private:
    const std::string& str;

    constexpr static char hex[17] = "0123456789abcdef";

    // Converts character code to hex to HEX
    static char to_hex(char code) { return hex[code & 15]; }
};

// Log string to JS console
void console_log(const std::string&);
