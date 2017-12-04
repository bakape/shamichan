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

// Render Links to different pages of the board index
// TODO: Pagination - total page count not yet exported
static Node render_pagination() { return { "span", "TODO" }; }

// Render a link to a catalog or board page
static Node render_catalog_link()
{
    return render_button(page->catalog ? "." : "catalog",
        lang->ui.at(page->catalog ? "return" : "catalog"), true);
}

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

    // File upload form
    if (page->board == "all" || !board_config->text_only) {
        form.children.push_back({
            "span",
            { { "class", "upload-container" } },
            {
                {
                    "span",
                    {},
                    {
                        {
                            "label",
                            {},
                            {
                                {
                                    "input",
                                    {
                                        { "type", "checkbox" },
                                        { "name", "spoiler" },
                                    },
                                },
                                { "span", lang->posts.at("spoiler") },
                            },
                        },
                    },
                },
                { "strong", { { "class", "upload-status" } } },
                { "br" },
                {
                    "input",
                    {
                        { "type", "file" },
                        { "name", "image" },
                        {
                            "accept",
                            "image/png, image/gif, image/jpeg, video/webm, "
                            "video/ogg, audio/ogg, application/ogg, video/mp4, "
                            "audio/mp4, audio/mp3, application/zip, "
                            "application/x-7z-compressed, application/x-xz, "
                            "application/x-gzip, audio/x-flac, text/plain",
                        },
                    },
                },
                { "br" },
            },
        });
    }

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
    auto& ch = aside_container.children;
    const Node cat_link = render_catalog_link();
    const Node pagination = render_pagination();
    ch.reserve(8);
    ch.push_back(render_thread_form());
    ch.push_back({
        "aside",
        {
            { "id", "refresh" },
            { "class", "act glass" },
        },
        { { "a", lang->ui.at("refresh") } },
    });
    ch.push_back(cat_link);
    ch.push_back(pagination);
    push_board_hover_info(ch);
    aside_container.write_html(s);

    s << "<hr>";

    // TODO: Render threads

    ch.clear();
    ch.push_back(cat_link);
    ch.push_back(pagination);

    // TODO: Render loading image

    aside_container.write_html(s);

    brunhild::set_inner_html("threads", s.str());
}

void render_board()
{
    // TODO: catalog
    render_index_page();
}
