#include "util.hh"
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
