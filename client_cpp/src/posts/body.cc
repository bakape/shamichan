#include "models.hh"
#include <string>
#include <string_view>
#include <vector>

using std::string_view;
using std::string;

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
