#include "util.hh"
#include "lang.hh"
#include <emscripten.h>
#include <string>
#include <string_view>
#include <tuple>

c_string_view get_inner_html(const std::string& id)
{
    return c_string_view((char*)EM_ASM_INT(
        {
            var s = document.getElementById(UTF8ToString($0)).innerHTML;
            var len = lengthBytesUTF8(s) + 1;
            var buf = Module._malloc(len);
            stringToUTF8(s, buf, len);
            return buf;
        },
        id.c_str()));
}

std::string pluralize(int n, const std::tuple<std::string, std::string>& word)
{
    std::string s;
    s.reserve(32);
    s += std::to_string(n) + ' ';
    switch (n) {
    case 1:
    case -1:
        s += std::get<0>(word);
        break;
    default:
        s += std::get<1>(word);
    }
    return s;
}

std::ostream& operator<<(std::ostream& os, const url_encode& u)
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
            os << '%' << url_encode::to_hex(ch >> 4)
               << url_encode::to_hex(ch & 15);
        }
    }
    return os;
}

brunhild::Children render_submit(bool cancel)
{
    brunhild::Children ch;
    ch.reserve(3);
    ch.push_back({
        "input",
        {
            { "type", "submit" },
            { "value", lang->ui.at("submit") },
        },
    });
    if (cancel) {
        ch.push_back({
            "input",
            {
                { "type", "button" },
                { "name", "cancel" },
                { "value", lang->ui.at("submit") },
            },
        });
    }
    ch.push_back({ "div", { { "class", "form-response admin" } } });
    return ch;
}

namespace console {

#define def_logger(key)                                                        \
    void key(const std::string& s)                                             \
    {                                                                          \
        EM_ASM_INT({ console.key(UTF8ToString($0)); }, s.c_str());             \
    }

def_logger(log);
def_logger(warn);
def_logger(error);
}
