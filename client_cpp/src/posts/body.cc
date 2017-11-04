#include "models.hh"
#include <string>
#include <string_view>
#include <vector>

using std::string_view;
using std::string;

// Splits string into string_views
static std::vector<string_view> split_string(
    const string& str, const string delimiter)
{
    std::vector<string_view> views;

    size_t pos = 0;
    size_t prev = 0;
    while ((pos = str.find(delimiter, prev)) != -1) {
        views.push_back({ &str[prev], pos - prev });
        prev = pos + 1;
    }

    // To get the last substring (or only, if delimiter is not found)
    views.push_back({ &str[prev], str.size() - pos });

    return views;
}

Node Post::render_body()
{
    Node n("blockquote");
    if (!body.size()) {
        return n;
    }
    state.reset(&n);

    bool first = true;
    for (auto&& line : split_string(body, "\n")) {
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
            continue;
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
    }

    return n;
}

// TODO
void Post::parse_temp_links(string_view frag)
{
    state.append({ "span", string(frag), true });
}

// TODO
void Post::parse_fragment(string_view frag)
{
    state.append({ "span", string(frag), true });
}
