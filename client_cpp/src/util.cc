#include <emscripten.h>
#include <string>

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
