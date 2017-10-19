#include <emscripten.h>
#include <string>
#include <tuple>

std::string convert_c_string(int str_p)
{
    std::string s((char*)str_p);
    delete[](char*) str_p;
    return s;
}

std::string get_inner_html(const std::string& id)
{
    return convert_c_string(EM_ASM_INT(
        {
            var s = document.getElementById(Pointer_stringify($0)).innerHTML;
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

void console_log(const std::string& s)
{
    EM_ASM_INT({ console.log(Pointer_stringify($0)); }, s.c_str());
}
