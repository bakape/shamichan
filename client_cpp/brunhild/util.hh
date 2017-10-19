#pragma once

#include <ostream>
#include <string>

namespace brunhild {

// Escape a user-submitted unsafe string s to protect against XSS and malformed
// HTML
// Example:
//		out << escape("this breaks layout </a>");
struct escape {
    escape(const std::string& s)
        : str(s)
    {
    }

    friend std::ostream& operator<<(std::ostream& os, const escape& e)
    {
        for (auto ch : e.str) {
            switch (ch) {
            case '&':
                os << "&amp;";
                break;
            case '\"':
                os << "&quot;";
                break;
            case '\'':
                os << "&apos;";
                break;
            case '<':
                os << "&lt;";
                break;
            case '>':
                os << "&gt;";
                break;
            case '`':
                os << "&#x60;";
                break;
            default:
                os << ch;
            }
        }
        return os;
    }

private:
    const std::string& str;
};
}
