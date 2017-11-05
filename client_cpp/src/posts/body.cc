#include "../lang.hh"
#include "../state.hh"
#include "models.hh"
#include <optional>
#include <string>
#include <string_view>
#include <tuple>

using std::string_view;
using std::string;
using std::optional;
using std::nullopt;
using std::tuple;
using std::get;

Node Post::render_body()
{
    Node n("blockquote");
    if (!body.size()) {
        return n;
    }
    state.reset(&n);

    bool first = true;
    parse_string(string_view(body), "\n",
        [this, &first](string_view line) {
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

            if (editing) {
                parse_code(
                    line, [this](string_view frag) { parse_temp_links(frag); });
            } else {
                parse_code(
                    line, [this](string_view frag) { parse_fragment(frag); });
            }

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
        },
        []() {});
    return n;
}

// Return, if b is a punctuation char
static bool is_punctuation(const char b)
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
static tuple<char, string_view, char> split_punctuation(const string_view word)
{
    tuple<char, string_view, char> re = { 0, word, 0 };

    // Split leading
    if (word.size() < 2) {
        return re;
    }
    if (is_punctuation(word[0])) {
        get<0>(re) = word[0];
        get<1>(re) = word.substr(1);
    }

    // Split trailing
    const size_t l = get<1>(re).size();
    if (l < 2) {
        return re;
    }
    if (is_punctuation(get<1>(re).back())) {
        get<2>(re) = get<1>(re).back();
        get<1>(re) = get<1>(re).substr(0, l - 1);
    }

    return re;
}

// Parses link to a post.
// If valid, returns number of extra '>' in front of the link and ID of the
// post, the link is pointing to.
static optional<tuple<int, uint64_t>> parse_post_link(string_view word)
{
    // Count leading '>'
    int count = 0;
    while (word.size()) {
        if (word[0] == '>') {
            count++;
            word = word.substr(1);
        } else {
            break;
        }
    }
    if (count < 2) {
        return nullopt;
    }
    count -= 2;

    // Verify everything else is digits
    if (!word.size()) {
        return nullopt;
    }
    int i = 0;
    while (i < word.size()) {
        if (word[i] < '0' || word[i] > '9') {
            return nullopt;
        }
        i++;
    }

    return { { count, std::stoull(string(word)) } };
}

// Render a temporary link for open posts
static Node render_temp_link(uint64_t id)
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
            { "class", "post-link temp" }, { "data-id", id_str },
            { "href", "#p" + id_str },
        },
        text,
    };
}

// Parse temporary links in open posts, that still may be edited
void Post::parse_temp_links(string_view frag)
{
    bool first = true;
    string buf;
    buf.reserve(frag.size());

    parse_string(frag, " ",
        [this, &first, &buf](auto frag) {
            if (!first) {
                buf += ' ';
            } else {
                first = false;
            }

            // Split leading and trailing punctuation, if any
            auto && [ lead_punct, word, trail_punct ] = split_punctuation(frag);
            if (lead_punct) {
                buf += lead_punct;
            }

            bool matched = false;
            if (word.size() && word[0] == '>') {
                if (auto l = parse_post_link(word); l) {
                    // Text preceding the link
                    auto[count, id] = *l;
                    for (int i = 0; i < count; i++) {
                        buf += '>';
                    }
                    state.append({ "span", buf, true });
                    buf.clear();

                    state.append(render_temp_link(id));
                    matched = true;
                }
            }
            if (!matched) {
                buf += word;
            }

            if (trail_punct) {
                buf += trail_punct;
            }
        },
        []() {});

    // Append any leftover text
    if (buf.size()) {
        state.append({ "span", buf, true });
    }
}

// TODO
void Post::parse_fragment(string_view frag)
{
    state.append({ "span", string(frag), true });
}
