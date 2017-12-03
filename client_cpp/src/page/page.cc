#include "../../brunhild/mutations.hh"
#include "../../brunhild/util.hh"
#include "../state.hh"
#include "../util.hh"
#include "board.hh"
#include "thread.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <optional>
#include <sstream>

using brunhild::Node;
using std::string;

void render_page()
{
    try {
        if (page->thread) {
            render_thread();
        } else {
            render_board();
        }
    } catch (const std::exception& ex) {
        console::error(ex.what());
    }
}

EMSCRIPTEN_BINDINGS(module_page)
{
    emscripten::function("render_page", &render_page);
}

string format_title(const string& board, const string& text)
{
    std::ostringstream s;
    s << '/' << board << "/ - " << brunhild::escape(text);
    return s.str();
}

void set_title(string t) { brunhild::set_inner_html("page-title", t); }

// Render notice widget, that reveals text on hover
Node render_hover_reveal(string tag, string label, string text)
{
    Node n{
        tag,
        { { "class", "hover-reveal" } },
        {
            { "span", { { "class", "act" } }, label },
            { "span", { { "class", "popup-menu glass" } }, text, true },
        },
    };
    if (tag == "aside") {
        *n.attrs["class"] += " glass";
    }
    return n;
}

// Renders a clickable button element.
// If href = std::nullopt, no href property is set on the link
Node render_button(std::optional<string> href, string text)
{
    Node a("a", text);
    if (href) {
        a.attrs["href"] = *href;
    }
    return { "span", { { "class", "act" } }, { a } };
}
