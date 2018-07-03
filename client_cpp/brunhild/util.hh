#pragma once

#include <string.h>
#include <string>
#include <type_traits>
#include <vector>

namespace brunhild {

// Escape a user-submitted unsafe string to protect against XSS and malformed
// HTML
std::string escape(const std::string& s);

// Allows returning the size of a std::string, std::string_view, char or char*
inline size_t string_size(const std::string& s) { return s.size(); }
inline size_t string_size(const std::string_view& s) { return s.size(); }
inline size_t string_size(char sep[[maybe_unused]]) { return 1; }
inline size_t string_size(const char* sep) { return strlen(sep); }

// Append-only rope data structure for more efficient HTML building
class Rope {
    template <class T> friend Rope& operator<<(Rope& r, const T& s);
    friend Rope& operator<<(Rope& r, const std::string& s);
    friend Rope& operator<<(Rope& r, std::string_view s);
    friend Rope& operator<<(Rope& r, char s);
    friend Rope& operator<<(Rope& r, const char* s);

public:
    Rope()
    {
        parts.reserve(16);
        // Calling .back() on an empty vector would be UB, so place one part
        // here
        parts.emplace_back().reserve(1 << 10);
    }

    // Dumps Rope contents to string
    std::string str()
    {
        size_t cap = 0;
        for (auto& s : parts) {
            cap += s.size();
        }
        std::string re;
        re.reserve(cap);
        for (auto& s : parts) {
            re += s;
        }
        return re;
    }

private:
    std::vector<std::string> parts;

    template <class T> Rope& append(const T& s)
    {
        std::string* last = &parts.back();
        if (last->size() + string_size(s) > last->capacity()) {
            const auto last_cap = last->capacity();
            last = &parts.emplace_back();
            last->reserve(last_cap << 1);
        }
        *last += s;
        return *this;
    }
};

// inline prevents these from colliding with the template during linking
inline Rope& operator<<(Rope& r, const std::string& s) { return r.append(s); }
inline Rope& operator<<(Rope& r, std::string_view s) { return r.append(s); }
inline Rope& operator<<(Rope& r, char s) { return r.append(s); }
inline Rope& operator<<(Rope& r, const char* s) { return r.append(s); }

// Append anything convertable with std::to_string() to Rope
template <class T> inline Rope& operator<<(Rope& r, const T& s)
{
    return r.append(std::to_string(s));
}
}
