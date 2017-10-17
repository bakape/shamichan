#include <cctype>
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
            var len = s.length + 1;
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

void pad(std::string& out, unsigned int n)
{
    if (n < 10) {
        out += '0';
    }
    out += std::to_string(n);
}

// Converts character code to hex to HEX
static char to_hex(char code)
{
    const static char hex[] = "0123456789abcdef";
    return hex[code & 15];
}

std::string url_encode(const std::string& s)
{
    std::string out;
    out.reserve(s.size() * 1.5);
    for (auto ch : s) {
        // Keep alphanumeric and other accepted characters intact
        if (isalnum(ch)) {
            out += ch;
            continue;
        }
        switch (ch) {
        case '-':
        case '_':
        case '.':
        case '~':
            out += ch;
            break;
        case ' ':
            out += '+';
            break;
        default:
            // Any other characters are percent-encoded.
            // Do not add the string together. That would just sum them as
            // integers.
            out += '%';
            out += to_hex(ch >> 4);
            out += to_hex(ch & 15);
        }
    }
    return out;
}

void console_log(const std::string& s)
{
    EM_ASM_INT({ console.log(Pointer_stringify($0)); }, s.c_str());
}
