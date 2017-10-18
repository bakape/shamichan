#include <emscripten.h>
#include <string>

using std::string;

void local_storage_set(const string key, const string val)
{
    EM_ASM_INT(
        {
            localStorage.setItem(Pointer_stringify(@0), Pointer_stringify(@1));
            return 0;
        },
        key.c_str(), val.c_str());
}

string local_storage_get(const string key)
{
    char* val = (char*)EM_ASM_INT(
        {
            var s = localStorage.getItem(Pointer_stringify($0)) || '';
            var len = lengthBytesUTF8(s) + 1;
            var buf = Module._malloc(len);
            stringToUTF8(s, buf, len);
            return buf;
        },
        key.c_str());
    const string s = string(val); // Coppies
    delete[] val;
    return s;
}
