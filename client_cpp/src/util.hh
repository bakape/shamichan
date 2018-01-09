#pragma once

#include "../brunhild/node.hh"
#include <cctype>
#include <functional>
#include <optional>
#include <ostream>
#include <string>
#include <string_view>
#include <tuple>

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
// If aside = true, renders the button as an <aside> element, instead of <span>.
brunhild::Node render_button(
    std::optional<std::string> href, std::string text, bool aside = false);

// Render a link to expand a thread
brunhild::Node render_expand_link(std::string board, unsigned long id);

// Render a link to only display the last 100  posts of a thread
brunhild::Node render_last_100_link(std::string board, unsigned long id);

// URL encode a string to pass into an ostream
struct url_encode {
    url_encode(const std::string& s)
        : str(s)
    {
    }

    friend std::ostream& operator<<(std::ostream& os, const url_encode& u);

private:
    const std::string& str;

    constexpr static char hex[17] = "0123456789abcdef";

    // Converts character code to hex to HEX
    static char to_hex(char code) { return hex[code & 15]; }
};

// Render submit button with and optional cancel button
brunhild::Children render_submit(bool cancel);

// Defers execution of a function, until a set amount of jobs are completed.
// Will dealocate itself, after all jobs are completed.
class WaitGroup {
public:
    // Defers execution of cb(), until done() has been called jobs times
    WaitGroup(unsigned int jobs, std::function<void()> cb)
        : jobs(jobs)
        , cb(cb)
    {
    }

    // Mark a jobs as completed
    void done()
    {
        if (--jobs == 0) {
            cb();
            delete this;
        }
    }

private:
    unsigned int jobs;
    std::function<void()> cb;
};

// Call the JS alert() function
void alert(std::string);

namespace console {
// Log string to JS console
void log(const std::string&);

// Log string to JS console as warning
void warn(const std::string&);

// Log string to JS console as error
void error(const std::string&);
}
