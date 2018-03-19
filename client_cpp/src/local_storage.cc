#include <emscripten.h>
#include <optional>
#include <string>

using std::string;

void local_storage_set(const string& key, const string& val)
{
    EM_ASM_INT(
        {
            localStorage.setItem(UTF8ToString($0), UTF8ToString($1));
            return 0;
        },
        key.c_str(), val.c_str());
}

std::optional<string> local_storage_get(const string& key)
{
    char* val = (char*)EM_ASM_INT(
        {
            var s = localStorage.getItem(UTF8ToString($0));
            if (!s) {
                return null;
            }
            var len = lengthBytesUTF8(s) + 1;
            var buf = Module._malloc(len);
            stringToUTF8(s, buf, len);
            return buf;
        },
        key.c_str());
    if (!val) {
        return {};
    }
    const string s = string(val); // Coppies
    delete[] val;
    return { s };
}
