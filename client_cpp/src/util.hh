#pragma once

#include "../brunhild/node.hh"
#include <cctype>
#include <functional>
#include <optional>
#include <ostream>
#include <sstream>
#include <string>
#include <string_view>
#include <tuple>
#include <vector>

// A string_view constructable from an owned char*, that also takes ownership
// of char* and frees it on drop.
class c_string_view : public std::string_view {
public:
    // Takes ownership of char*. char* must not be NULL.
    c_string_view(char* s)
        : std::string_view(s)
        , ch(s)
    {
    }

    // Must prevent ownership of ch from being copied
    c_string_view(c_string_view&&) = default;
    c_string_view(const c_string_view&) = delete;

    ~c_string_view() { delete[] ch; }

    // Return subview between start and end indices.
    // Returned string_view is only valid for the lifetime of this
    // c_string_view.
    std::string_view substr(size_t start = 0, size_t end = npos) const
    {
        return std::string_view(ch).substr(start, end);
    }

private:
    char* ch;
};

// Read inner HTML from DOM element by ID
c_string_view get_inner_html(const std::string& id);

// Return either the singular or plural form of a translation, depending on n.
// word is the index used for finding the localization tuple.
std::string pluralize(int n, std::string word);

// Renders a clickable button element.
// If href = std::nullopt, no href property is set on the link.
brunhild::Node render_button(std::optional<std::string> href, std::string text);

// Render a link to expand a thread
brunhild::Node render_expand_link(std::string board, unsigned long id);

// Render a link to only display the last 100  posts of a thread
brunhild::Node render_last_100_link(std::string board, unsigned long id);

// URL encode a string to pass into an ostream
struct url_encode {
    url_encode(const std::string& s);
    friend std::ostream& operator<<(std::ostream& os, const url_encode& u);

private:
    const std::string& str;

    // Converts character code to hex to HEX
    static char to_hex(char code);
};

// Render submit button with and optional cancel button
brunhild::Children render_submit(bool cancel);

// Defers execution of a function, until a set amount of jobs are completed.
// Will dealocate itself, after all jobs are completed.
class WaitGroup {
public:
    // Defers execution of cb(), until done() has been called jobs times
    WaitGroup(unsigned jobs, std::function<void()> cb)
        : jobs(jobs)
        , cb(cb)
    {
    }

    // Mark a jobs as completed
    void done()
    {
        if (--jobs == 0) {
            cb();
            this->~WaitGroup();
        }
    }

private:
    unsigned jobs;
    std::function<void()> cb;
    ~WaitGroup() = default;
};

// Call the JS alert() function
void alert(std::string);

// Convert string to lowercase
std::string to_lower(const std::string&);

// Run function an all parts of string-like T split by separator sep
template <class T, class U>
inline void split_string(
    T frag, U sep, std::function<void(std::string_view)> on_frag)
{
    auto _frag = std::string_view(frag);
    size_t i;
    const size_t sep_s = brunhild::string_size(sep);
    while (1) {
        i = _frag.find(sep);
        on_frag(_frag.substr(0, i));
        if (i != std::string::npos) {
            _frag = _frag.substr(i + sep_s);
        } else {
            break;
        }
    }
}

// Join iterable container into string.
// with_space: insert space after comma
template <class T>
inline std::string join_to_string(T cont, bool with_space = false)
{
    std::ostringstream s;
    bool first = true;
    for (auto& item : cont) {
        if (!first) {
            s << ',';
            if (with_space) {
                s << ' ';
            }
        } else {
            first = false;
        }
        s << item;
    }
    return s.str();
}

namespace console {
// Log string to JS console
void log(const std::string&);

// Log string to JS console as warning
void warn(const std::string&);

// Log string to JS console as error
void error(const std::string&);
}
