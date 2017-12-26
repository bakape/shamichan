#include "../../brunhild/mutations.hh"
#include "../lang.hh"
#include "../posts/etc.hh"
#include "../posts/models.hh"
#include "../state.hh"
#include "../util.hh"
#include "page.hh"
#include <algorithm>
#include <optional>
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>

using brunhild::Node;
using std::ostringstream;
using std::string;

// Modes for sorting threads
enum class SortMode { bump, last_reply, creation, reply_count, file_count };

// Current thread sorting mode
SortMode sort_mode = SortMode::bump;

// Generate a sorted vector of thread references according to current sorting
// mode
static std::vector<Thread*> sort_threads()
{
    std::vector<Thread*> t;
    t.reserve(threads->size());
    for (auto & [ _, thread ] : *threads) {
        t.push_back(&thread);
    }

    const bool is_all = page->board == "all";
    std::sort(t.begin(), t.end(), [=](auto a, auto b) {
        if (is_all || (a->sticky && b->sticky)) {
            switch (sort_mode) {
            case SortMode::bump:
                return a->bump_time < b->bump_time;
            case SortMode::last_reply:
                return a->reply_time < b->reply_time;
            case SortMode::creation:
                return a->time < b->time;
            case SortMode::reply_count:
                return a->post_ctr < b->post_ctr;
            case SortMode::file_count:
                return a->image_ctr < b->image_ctr;
            }
        }
        if (a->sticky) {
            return false;
        }
        if (b->sticky) {
            return true;
        }
    });

    return t;
}

// Render threads on a board page
static void render_index_threads(ostringstream& s)
{
    // Group all posts by thread. These are already sorted by post ID.
    std::unordered_map<unsigned long, std::vector<Post*>> by_thread;
    by_thread.reserve(threads->size());
    for (auto & [ _, p ] : *posts) {
        by_thread[p.op].push_back(&p);
    }

    sort_mode = SortMode::bump;
    s << "<div id=index-thread-container>";
    const auto sorted = sort_threads();
    for (int i = 0; i < sorted.size(); i++) {
        const auto t = sorted[i];

        s << "<section class=\"index-thread";
        if (t->deleted) {
            s << " deleted";
        }
        s << "\">";
        if (t->deleted) {
            delete_toggle.write_html(s);
        }

        for (auto p : by_thread[t->id]) {
            p->init();
            p->write_html(s);
        }

        if (i != sorted.size() - 1) {
            s << "<hr>";
        }
        s << "</section>";
    }
    s << "</div><hr>";
}

// Render Links to different pages of the board index
// TODO: Pagination - total page count not yet exported
static Node render_pagination()
{
    const unsigned int n = page->page;
    const unsigned int total = page->page_total;
    ostringstream s;

    auto link = [&s](unsigned int i, string text) {
        s << "<a href=\"?page=" << i << "\">" << text << "</a>";
    };

    if (n) {
        if (n - 1) {
            link(0, "<<");
        }
        link(n - 1, "<");
    }
    for (unsigned int i = 0; i < total; i++) {
        if (i != n) {
            link(i, std::to_string(i));
        } else {
            s << "<b>" << i << "</b>";
        }
    }
    if (n != total - 1) {
        link(n + 1, ">");
        if (n + 1 != total - 1) {
            link(total - 1, ">>");
        }
    }

    return { "aside", { { "class", "glass spaced" } }, s.str() };
}

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
                { "required", "" },
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
                { "hidden", "" },
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
                        attrs["checked"] = "";
                        attrs["disabled"] = "";
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
    if (!page->catalog) {
        ch.push_back(pagination);
    }
    push_board_hover_info(ch);
    aside_container.write_html(s);

    s << "<hr>";

    render_index_threads(s);

    ch.clear();
    ch.push_back(cat_link);
    if (!page->catalog) {
        ch.push_back(pagination);
    }

    // TODO: Render loading image

    aside_container.write_html(s);

    brunhild::set_inner_html("threads", s.str());
}

void render_board()
{
    // TODO: catalog
    render_index_page();
}
