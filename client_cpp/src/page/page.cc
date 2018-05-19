#include "page.hh"
#include "../../brunhild/mutations.hh"
#include "../../brunhild/util.hh"
#include "../connection/sync.hh"
#include "../lang.hh"
#include "../posts/hide.hh"
#include "../state.hh"
#include "../util.hh"
#include "board.hh"
#include "scroll.hh"
#include "thread.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <optional>
#include <sstream>

using brunhild::Node;
using std::string;

void render_page()
{
    recurse_hidden_posts();
    page_view.patch();

    if (page.post) {
        scroll_to_post(page.post);
    } else {
        const string s
            = emscripten::val::global("location")["hash"].as<string>();
        if (s == "#top" || s == "#bottom") {
            brunhild::scroll_into_view(s.substr(1));
        }
    }

    // TODO: Hide loading image
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
static Node render_hover_reveal(string tag, string label, string text)
{
    Node n{
        tag, { { "class", "hover-reveal" } },
        {
            { "span", { { "class", "act" } }, label },
            { "span", { { "class", "popup-menu glass" } }, text, true },
        },
    };
    if (tag == "aside") {
        n.attrs["class"] += " glass";
    }
    return n;
}

void push_board_hover_info(brunhild::Children& ch)
{
    const char* tag = page.thread ? "span" : "aside";
    if (board_config.notice != "") {
        ch.push_back(render_hover_reveal(
            tag, board_config.notice, lang.ui.at("showNotice")));
    }
    if (board_config.rules != "") {
        ch.push_back(
            render_hover_reveal(tag, board_config.rules, lang.ui.at("rules")));
    }
}

Node ImageBanner::render()
{
    Node n("h1", { { "class", "image-banner" } });

    auto const& b = board_config.banners;
    if (page.thread || !b.size()) {
        n.hide();
        return n;
    }

    Node ch;
    const int i = rand() % b.size();
    if (b[i] == FileType::webm) {
        ch.tag = "video";
        ch.attrs = { { "autoplay", "" }, { "loop", "" } };
    } else {
        ch.tag = "img";
    }
    std::ostringstream s;
    s << "/assets/banners/" << page.board << '/' << i;
    ch.attrs["src"] = s.str();
    n.children = { ch };

    return n;
}

Node PageTitle::render()
{
    Node n("h1");
    if (page.thread) {
        n.hide();
        return n;
    }
    n.inner_html = { format_title(page.board, boards[page.board]) };
    return n;
}

PageView::PageView()
    : CompositeView<brunhild::VirtualView>("section", "threads")
{
}

std::vector<brunhild::VirtualView*> PageView::get_list()
{
    return { new ImageBanner(), new PageTitle() };
}
