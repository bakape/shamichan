#include <string>

namespace brunhild {

std::string escape(const std::string& s)
{
    std::string out;
    out.reserve(s.size() * 1.1);
    for (auto ch : s) {
        switch (ch) {
        case '&':
            out += "&amp;";
            break;
        case '\"':
            out += "&quot;";
            break;
        case '\'':
            out += "&apos;";
            break;
        case '<':
            out += "&lt;";
            break;
        case '>':
            out += "&gt;";
            break;
        case '`':
            out += "&#x60;";
            break;
        case '\0':
            break;
        default:
            out += ch;
        }
    }
    return out;
}
}
