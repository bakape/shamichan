#include "../../brunhild/mutations.hh"
#include "../../brunhild/view.hh"
#include "../form.hh"
#include "../lang.hh"
#include "../local_storage.hh"
#include "../page/page.hh"
#include "../state.hh"
#include <sstream>
#include <vector>

using brunhild::Children;
using brunhild::Node;

static bool selection_open = false;

// Returns, if board links should point to catalog pages
static bool point_to_catalog()
{
    const auto s = local_storage_get("pointToCatalog");
    if (!s) {
        return false;
    }
    return *s == "true";
}

class BoardNavigation : public brunhild::View {
public:
    // Not in constructor, so we can allocate it to static memory
    void init();

    Node render()
    {
        std::ostringstream s;
        const bool catalog = point_to_catalog();
        s << '[';
        board_link(s, "all", catalog);
        for (auto& b : *boards) {
            s << " / ";
            board_link(s, b, catalog);
        }
        s << "] [<a class=\"board-selection bold mono\">"
          << (selection_open ? "-" : "+") << "</a>]";
        return { "nav", { { "id", "board-navigation" } }, s.str() };
    }

private:
    // Renders a link to a board
    void board_link(
        std::ostringstream& s, const std::string& board, const bool catalog)
    {
        s << "<a href=\"../" << board << '/';
        if (catalog) {
            s << "catalog";
        }
        s << "\">" << board << "</a>";
    }
};

class BoardSelectionForm : public Form {
public:
    // Not in constructor, so we can allocate it to static memory
    void init()
    {
        Form::init();
        brunhild::append("left-panel", html());
        // TODO: Event handlers
    }

    void remove() override;

protected:
    Node render_inputs() override
    {
        return {
            "div",
            {},
            {
                {
                    "input",
                    {
                        { "type", "text" },
                        { "class", "full-width" },
                        { "name", "search" },
                        { "placeholder", lang->ui.at("search") },
                    },
                },
                { "br" },
            },
        };
    }

    void on_submit(emscripten::val e) override
    {
        // TODO
    }

    Node render_footer() override
    {
        Children ch;
        ch.reserve(board_titles->size());
        for (auto & [ board, title ] : *board_titles) {
            ch.push_back({
                "label",
                {},
                {
                    {
                        "input",
                        {
                            { "type", "checkbox" },
                            { "name", board },
                        },
                    },
                    {
                        "a",
                        { { "href", '/' + board + '/' } },
                        format_title(board, title),
                    },
                    { "br" },
                },
            });
        }
        return { "div", {}, ch };
    }

    Children render_after_controls() override
    {
        return {
            {
                "label",
                {},
                {
                    {
                        "input",
                        {
                            { "type", "checkbox" },
                            { "name", "pointToCatalog" },
                        },
                        lang->ui.at("pointToCatalog"),
                    },
                },
            },
        };
    }
};

static BoardNavigation bn;
static BoardSelectionForm bsf;

void BoardNavigation::init()
{
    // TODO: Remove, when server-side templates ported
    brunhild::remove("board-navigation");

    View::init();
    on("click", ".board-selection", [this](auto& _) {
        console::log("running handler");
        if (selection_open) {
            bsf.remove();
        } else {
            selection_open = true;
            bsf.init();
        }
        patch();
    });
    brunhild::append("banner", html());
}

void BoardSelectionForm::remove()
{
    Form::remove();
    selection_open = false;
    bn.patch();
}

void init_top_header() { bn.init(); }
