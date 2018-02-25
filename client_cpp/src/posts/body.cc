#include "../lang.hh"
#include "../state.hh"
#include "etc.hh"
#include "models.hh"
#include "url.hh"
#include <cctype>
#include <optional>
#include <string>
#include <string_view>
#include <tuple>
#include <type_traits>

using std::function;
using std::nullopt;
using std::optional;
using std::string;
using std::string_view;
using std::tuple;

// Allows returning the size of a char or char*
template <class D> size_t sep_size(D sep) { return sep.size(); }
template <> size_t sep_size<char>(char sep[[maybe_unused]]) { return 1; }
template <> size_t sep_size<const char*>(const char* sep)
{
    return strlen(sep);
}

// Split string_view into subviews on delimiter D, call on_frag on each
// fragment and call on_match after each matched delimiter
template <class D>
void parse_string(string_view frag, D sep, function<void(string_view)> on_frag,
    function<void()> on_match = []() {})
{
    size_t i;
    const size_t sep_s = sep_size(sep);
    while (1) {
        i = frag.find(sep);
        on_frag(frag.substr(0, i));
        if (i != string::npos) {
            frag = frag.substr(i + sep_s);
            on_match();
        } else {
            break;
        }
    }
}

Node Post::render_body()
{
    Node n("blockquote");
    if (!body.size()) {
        return n;
    }
    state.reset(&n);

    bool first = true;
    parse_string(string_view(body), '\n', [this, &first](string_view line) {
        state.quote = false;

        // Prevent successive empty lines
        if (!first) {
            if (state.successive_newlines < 2) {
                state.append({ "br" });
            }
        } else {
            first = false;
        }
        if (!line.size()) {
            state.successive_newlines++;
            return;
        }

        state.successive_newlines = 0;
        if (line[0] == '>') {
            state.quote = true;
            state.append({ "em" }, true);
        }
        if (state.spoiler) {
            state.append({ "del" }, true);
        }
        if (state.bold) {
            state.append({ "b" }, true);
        }
        if (state.italic) {
            state.append({ "i" }, true);
        }

        parse_code(line, [this](string_view frag) {
            editing ? parse_temp_links(frag) : parse_fragment(frag);
        });

        // Close any unclosed tags
        if (state.italic) {
            state.ascend();
        }
        if (state.bold) {
            state.ascend();
        }
        if (state.spoiler) {
            state.ascend();
        }
        if (state.quote) {
            state.ascend();
        }
    });
    return n;
}

void Post::parse_code(string_view frag, Post::OnFrag fn)
{
    parse_string(frag, "``",
        [this, fn](string_view frag) {
            if (state.code) {
                // Strip quotes
                size_t num_quotes = 0;
                while (frag.size() && frag[0] == '>') {
                    frag = frag.substr(1);
                }
                if (num_quotes) {
                    string s;
                    s.reserve(4 * num_quotes);
                    for (size_t i = 0; i <= num_quotes; i++) {
                        s += "&gt;";
                    }
                    state.append({ "span", s });
                }

                highlight_syntax(frag);
            } else {
                parse_spoilers(frag, fn);
            }
        },
        [this]() { state.code = !state.code; });
}

void Post::parse_spoilers(string_view frag, Post::OnFrag fn)
{
    parse_string(frag, "**",
        [this, fn](string_view frag) { parse_bolds(frag, fn); },
        [this]() {
            if (state.italic) {
                state.ascend();
            }
            if (state.bold) {
                state.ascend();
            }

            if (state.spoiler) {
                state.ascend();
            } else {
                state.append({ "del" }, true);
            }

            if (state.bold) {
                state.append({ "b" }, true);
            }
            if (state.italic) {
                state.append({ "i" }, true);
            }

            state.spoiler = !state.spoiler;
        });
}

void Post::parse_bolds(string_view frag, Post::OnFrag fn)
{
    parse_string(frag, "__",
        [this, fn](string_view frag) { parse_italics(frag, fn); },
        [this]() {
            if (state.italic) {
                state.ascend();
            }

            if (state.bold) {
                state.ascend();
            } else {
                state.append({ "b" }, true);
            }

            if (state.italic) {
                state.append({ "i" }, true);
            }

            state.bold = !state.bold;
        });
}

void Post::parse_italics(string_view frag, Post::OnFrag fn)
{
    parse_string(frag, "~~", fn, [this]() {
        if (state.italic) {
            state.ascend();
        } else {
            state.append({ "i" }, true);
        }

        state.italic = !state.italic;
    });
}

// Return, if b is a punctuation char
static inline bool is_punctuation(const char b)
{
    switch (b) {
    case '!':
    case '"':
    case '\'':
    case '(':
    case ')':
    case ',':
    case '-':
    case '.':
    case ':':
    case ';':
    case '?':
    case '[':
    case ']':
        return true;
    default:
        return false;
    }
}

// Splits off one byte of leading and trailing punctuation, if any, and returns
// the 3 split parts. If there is no edge punctuation, the respective char
// is null.
static inline tuple<char, string_view, char> split_punctuation(string_view word)
{
    char lead = 0, trail = 0;
    if (word.size() > 1 && is_punctuation(word[0])) {
        lead = word[0];
        word = word.substr(1);
    }
    if (word.size() > 1 && is_punctuation(word.back())) {
        trail = word.back();
        word = word.substr(0, word.size() - 1);
    }
    return { lead, word, trail };
}

void Post::parse_words(string_view frag, Post::OnFrag fn)
{
    bool first = true;
    state.buf.reserve(frag.size());

    parse_string(frag, " ", [this, &first, fn](string_view frag) {
        if (!first) {
            state.buf += ' ';
        } else {
            first = false;
        }

        // Split leading and trailing punctuation, if any
        auto[lead_punct, word, trail_punct] = split_punctuation(frag);
        if (lead_punct) {
            state.buf += lead_punct;
        }
        fn(word);
        if (trail_punct) {
            state.buf += trail_punct;
        }
    });

    // Append any leftover text
    state.flush_text();
}

// Strip leading '>' and return stripped count
static int strip_gt(string_view& word)
{
    int count = 0;
    while (word.size()) {
        if (word[0] == '>') {
            count++;
            word = word.substr(1);
        } else {
            break;
        }
    }
    return count;
}

// Parses link to a post.
// If valid, returns number of extra '>' in front of the link and ID of the
// post, the link is pointing to.
static optional<tuple<int, unsigned long>> parse_post_link(string_view word)
{
    // Count leading '>'
    int count = strip_gt(word);
    if (count < 2) {
        return nullopt;
    }
    count -= 2;

    // Verify everything else is digits
    if (!word.size()) {
        return nullopt;
    }
    for (char ch : word) {
        if (!isdigit(ch)) {
            return nullopt;
        }
    }

    return { { count, std::stoull(string(word)) } };
}

// Render a temporary link for open posts
static Node render_temp_link(unsigned long id)
{
    const string id_str = std::to_string(id);
    string text = ">>" + id_str;
    if (post_ids->mine.count(id)) {
        text += ' ';
        text += lang->posts.at("you");
    }
    return {
        "a",
        {
            { "class", "post-link temp" },
            { "data-id", id_str },
            { "href", "#p" + id_str },
        },
        text,
    };
}

// Parse temporary links in open posts, that still may be edited
void Post::parse_temp_links(string_view frag)
{
    parse_words(frag, [this](string_view word) {
        bool matched = false;
        if (word.size() && word[0] == '>') {
            if (auto l = parse_post_link(word)) {
                // Text preceding the link
                auto[count, id] = *l;
                state.append(render_temp_link(id), false, count);
                matched = true;
            }
        }
        if (!matched) {
            state.buf += word;
        }
    });
}

// Parse a line fragment of a closed post
void Post::parse_fragment(string_view frag)
{
    parse_words(frag, [this](string_view word) {
        if (!word.size()) {
            return;
        }

        bool matched = false;
        switch (word[0]) {
        case '>':
            // Post links
            if (auto l = parse_post_link(word)) {
                auto[count, id] = *l;

                // In case the server parsed this differently.
                // Maybe older version.
                if (links.count(id)) {
                    state.append(render_post_link(id, links[id]), false, count);
                    matched = true;
                    break;
                }
            }

            // Internal and custom reference URLs
            if (auto l = parse_reference(word)) {
                auto[count, n] = *l;
                state.append(n, false, count);
                matched = true;
            }
            break;
        case '#':
            // Hash commands
            if (auto n = parse_commands(word)) {
                state.append(*n);
                matched = true;
            }
            break;
        default:
            // Generic HTTP(S)/FTP(S) URLs, magnet links and embeds
            if (auto n = parse_url(word)) {
                state.append(*n);
                matched = true;
            }
        }
        if (!matched) {
            state.buf += word;
        }
    });
}

optional<tuple<int, Node>> Post::parse_reference(string_view word)
{
    int gts = strip_gt(word);
    if (gts < 3) {
        return nullopt;
    }
    gts -= 3;

    // Strip '/'
    if (word.size() < 3 || word.front() != '/' || word.back() != '/') {
        return nullopt;
    }
    word = word.substr(1, word.size() - 2);

    // Verify the rest is alphanumerics
    for (char ch : word) {
        if (!isalnum(ch)) {
            return nullopt;
        }
    }

    string href;
    const string s(word);
    if (boards->count(s)) { // Linking a board
        href.reserve(s.size() + 2);
        href = '/' + s + '/';
    } else if (config->links.count(s)) { // Custom external URL
        href = config->links.at(s);
    } else {
        return nullopt;
    }

    string text;
    text.reserve(word.size() + 5);
    text = ">>>/" + s + "/";

    return { { gts, render_link(string_view(href), text) } };
}
