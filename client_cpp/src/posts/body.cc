#include "view.hh"
#include <string_view>

// TODO: Also use string view for string FFI, where possible

using std::string_view;

Node PostView::render_body(const Post& p)
{
    Node n("blockquote");

    size_t next = -1;
    size_t i = 0;
    do {
        i = next + 1;
        next = p.body.find_first_of('\n', i);
        string_view line(&p.body[i], next - i);

        state.quote = false;

        // Prevent successive empty lines
        if (i && state.successive_newlines < 2) {
            n.children.push_back({ "br" });
        }
        if (!line.size()) {
            state.successive_newlines++;
            continue;
        }

        state.successive_newlines = 0;
        if (line[0] == '>') {
            state.quote = true;
        }

    } while (next != -1);

    return n;
}
