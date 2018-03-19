#include "util.hh"
#include "lang.hh"
#include <emscripten.h>
#include <locale>
#include <optional>
#include <sstream>
#include <string>
#include <string_view>
#include <tuple>

using brunhild::Node;
using std::string;

c_string_view get_inner_html(const string& id)
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

string pluralize(int n, string word)
{
    std::string s;
    s.reserve(32);
    s = std::to_string(n) + ' ';

    auto& ln = lang.plurals.at(word);
    switch (n) {
    case 1:
    case -1:
        s += std::get<0>(ln);
        break;
    default:
        s += std::get<1>(ln);
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
            { "type", "submit" }, { "value", lang.ui.at("submit") },
        },
    });
    if (cancel) {
        ch.push_back({
            "input",
            {
                { "type", "button" }, { "name", "cancel" },
                { "value", lang.ui.at("submit") },
            },
        });
    }
    ch.push_back({ "div", { { "class", "form-response admin" } } });
    return ch;
}

Node render_button(std::optional<string> href, string text, bool aside)
{
    Node a("a", text);
    if (href) {
        a.attrs["href"] = *href;
    }
    string cls = "act";
    if (aside) {
        cls += " glass";
    }
    return { aside ? "aside" : "span", { { "class", cls } }, { a } };
}

Node render_expand_link(string board, unsigned long id)
{
    std::ostringstream s;
    s << '/' << board << '/' << id;
    return render_button(s.str(), lang.posts.at("expand"));
}

Node render_last_100_link(string board, unsigned long id)
{
    std::ostringstream s;
    s << '/' << board << '/' << id << "?last=100#bottom";
    return render_button(s.str(), lang.ui.at("last") + " 100");
}

void alert(std::string msg)
{
    EM_ASM_INT({ alert(UTF8ToString($0)); }, msg.c_str());
}

std::string to_lower(const std::string& s)
{
    auto loc = std::locale();
    std::string conv;
    conv.reserve(s.size());
    for (auto ch : s) {
        conv += std::tolower(ch, loc);
    }
    return conv;
}

namespace console {

#define DEF_LOGGER(key)                                                        \
    void key(const string& s)                                                  \
    {                                                                          \
        EM_ASM_INT({ console.key(UTF8ToString($0)); }, s.c_str());             \
    }

DEF_LOGGER(log)
DEF_LOGGER(warn)
DEF_LOGGER(error)
}
