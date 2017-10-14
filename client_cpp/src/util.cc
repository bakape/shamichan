#include <emscripten.h>
#include <string>
#include <tuple>

std::string get_inner_html(const std::string& id)
{
    char* val = (char*)EM_ASM_INT(
        {
            var s = document.getElementById(Pointer_stringify($0)).innerHTML;
            var len = s.length + 1;
            var buf = Module._malloc(len);
            stringToUTF8(s, buf, len);
            return buf;
        },
        id.c_str());
    auto s = std::string(val); // Coppies
    delete[] val;
    return s;
}

std::string pluralize(int n, const std::tuple<std::string, std::string>& word)
{
    auto s = std::to_string(n) + ' ';
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
