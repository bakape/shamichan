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
        case '\'':
            out += "&#39;"; // "&#39;" is shorter than "&apos;"
            break;
        case '<':
            out += "&lt;";
            break;
        case '>':
            out += "&gt;";
            break;
        case '\"':
            out += "&#34;"; // "&#34;" is shorter than "&quot;"
            break;
        default:
            out += ch;
        }
    }
    return out;
}
}
