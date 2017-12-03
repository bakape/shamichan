#include "../../brunhild/mutations.hh"
#include "../lang.hh"
#include "../state.hh"
#include "../util.hh"
#include "page.hh"
#include <optional>
#include <sstream>
#include <string>

using brunhild::Node;
using std::ostringstream;
using std::string;

// Render form for creating new threads
static Node render_thread_form()
{
    Node form("form",
        {
            { "id", "new-thread-form" },
            { "action", "/api/create-thread" },
            { "method", "post" },
            { "enctype", "multipart/form-data" },
            { "class", "hidden" },
        });
    form.children.reserve(10);

    // Board selection input
    if (page->board == "all") {
        Node sel("select",
            {
                { "name", "board" },
                { "required", std::nullopt },
            });
        sel.children.reserve(board_titles->size());
        for (auto & [ board, title ] : *board_titles) {
            sel.children.push_back({
                "option",
                {
                    { "value", board },
                },
                format_title(board, title),
            });
        }

        form.children.push_back(sel);
        form.children.push_back({ "br" });
    } else {
        form.children.push_back({
            "input",
            {
                { "type", "text" },
                { "name", "board" },
                { "value", page->board },
                { "hidden", std::nullopt },
            },
        });
    }

    // Live post editing toggle for thread
    auto & [ label, title ] = lang->forms.at("nonLive");
    form.children.push_back({
        "label",
        { { "title", title } },
        {
            {
                "input",
                []() {
                    brunhild::Attrs attrs({
                        { "type", "checkbox" },
                        { "name", "nonLive" },
                    });
                    if (board_config->non_live) {
                        attrs["checked"] = std::nullopt;
                        attrs["disabled"] = std::nullopt;
                    }
                    return attrs;
                }(),
            },
            label,
        },
    });
    form.children.push_back({ "br" });

    // TODO: Upload form
    // TODO: Captcha

    auto submit = render_submit(true);
    form.children.insert(form.children.end(), submit.begin(), submit.end());

    return {
        "aside",
        {
            { "id", "thread-form-container" },
            { "class", "glass" },
        },
        // Disambiguate constructor
        brunhild::Children({
            render_button(std::nullopt, lang->ui.at("newThread")),
            form,
        }),
    };
}

// Render board index page
static void render_index_page()
{
    ostringstream s;

    // Render a random banner, if any
    if (auto const& b = board_config->banners; b.size()) {
        s << "<h1 class=image-banner>";
        const int i = rand() % b.size();
        if (b[i] == FileType::webm) {
            s << "<video autoplay loop";
        } else {
            s << "<img";
        }
        s << " src=\"/assets/banners/" << page->board << '/' << i << "\"></h1>";
    }

    const string title = format_title(page->board, board_config->title);
    s << "<h1 id=page-title>" << title << "</h1>";
    set_title(title);

    Node aside_container("span", { { "class", "aside-container" } });
    aside_container.children.reserve(8);
    aside_container.children.push_back(render_thread_form());

    // TODO: The rest

    aside_container.write_html(s);

    brunhild::set_inner_html("threads", s.str());
}

void render_board()
{
    // TODO: catalog
    render_index_page();
}
