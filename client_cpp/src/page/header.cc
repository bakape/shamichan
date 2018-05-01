#include "../../brunhild/mutations.hh"
#include "../../brunhild/view.hh"
#include "../form.hh"
#include "../lang.hh"
#include "../local_storage.hh"
#include "../page/page.hh"
#include "../state.hh"
#include "../util.hh"
#include <memory>
#include <set>
#include <sstream>

using nlohmann::json;
using brunhild::Children;
using brunhild::Node;

// Default ordering is ascending
std::set<std::string, std::greater<std::string>> selected_boards;

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
    BoardNavigation();
    Node render();

private:
    // Renders a link to a board
    void board_link(
        std::ostringstream& s, const std::string& board, bool catalog)
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
    BoardSelectionForm();
    void remove() override;

protected:
    Node render_inputs() override
    {
        return {
            "div", {},
            {
                {
                    "input",
                    {
                        { "type", "text" }, { "class", "full-width" },
                        { "name", "search" },
                        { "placeholder", lang.ui.at("search") },
                    },
                },
                { "br" },
            },
        };
    }

    Node render_footer() override
    {
        Children ch;
        ch.reserve(boards.size());
        const bool to_catalog = point_to_catalog();
        std::ostringstream s;
        for (auto & [ board, title ] : boards) {
            s.str("");
            s << '/' << board << '/';
            if (to_catalog) {
                s << "catalog";
            }

            brunhild::Attrs attrs
                = { { "type", "checkbox" }, { "name", board } };
            if (selected_boards.count(board)) {
                attrs["checked"] = "";
            }

            bool display = true;
            if (filter.size()) {
                display = board.find(filter) != std::string::npos
                    || to_lower(title).find(filter) != std::string::npos;
            }

            ch.push_back({
                "label", { { "class", display ? "" : "hidden" } },
                {
                    { "input", attrs },
                    {
                        "a",
                        // Need to copy to  prevent invalidating on reset
                        { { "href", std::string(s.str()) } },
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
        brunhild::Attrs attrs
            = { { "type", "checkbox" }, { "name", "pointToCatalog" } };
        if (point_to_catalog()) {
            attrs["checked"] = "";
        }
        return {
            {
                "label", {},
                { { "input", attrs, lang.ui.at("pointToCatalog") } },
            },
        };
    }

private:
    std::string filter;
};

static std::unique_ptr<BoardNavigation> bn;
static std::unique_ptr<BoardSelectionForm> bsf;

Node BoardNavigation::render()
{
    std::ostringstream s;
    const bool catalog = point_to_catalog();
    s << '[';
    board_link(s, "all", catalog);
    for (auto & [ b, _ ] : boards) {
        if (!selected_boards.count(b)) {
            continue;
        }
        s << " / ";
        board_link(s, b, catalog);
    }
    s << "] [<a class=\"board-selection bold mono\">" << (bsf ? "-" : "+")
      << "</a>]";
    return { "nav", { { "id", "board-navigation" } }, s.str() };
}

// Read selected boards from localStorage
static void read_selected()
{
    selected_boards.clear();
    if (auto s = local_storage_get("selectedBoards"); s) {
        split_string(*s, ',', [](std::string_view s) {
            auto str = std::string(s);
            if (boards.count(str)) {
                selected_boards.insert(str);
            }
        });
    }
}

BoardNavigation::BoardNavigation()
    : View("board-navigation")
{
    // TODO: Remove, when server-side templates ported
    brunhild::remove("board-navigation");

    read_selected();
    View::init();
    on("click", ".board-selection", [this](auto& _) {
        if (bsf) {
            bsf->remove();
        } else {
            bsf.reset(new BoardSelectionForm());
        }
        patch();
    });
    brunhild::append("banner", html());
}

BoardSelectionForm::BoardSelectionForm()
    : Form({}, true)
{
    // Need to reduce any chance conflicts between multiple tabs
    read_selected();
    bn->patch();

    Form::init();

    on("input", "input[name=search]", [this](auto& event) {
        filter = event["target"]["value"].template as<std::string>();
        patch();
    });

    // Add or remove board to selected board for display or toggle catalog
    // linking
    on("change", "input[type=checkbox]", [this](auto& e) {
        auto name = e["target"]["name"].template as<std::string>();
        bool checked = e["target"]["checked"].template as<bool>();

        if (name == "pointToCatalog") {
            local_storage_set("pointToCatalog", checked ? "true" : "false");
            patch();
        } else {
            if (checked) {
                selected_boards.insert(name);
            } else {
                selected_boards.erase(name);
            }
            local_storage_set(
                "selectedBoards", join_to_string(selected_boards));
        }

        bn->patch();
    });

    brunhild::append("left-panel", html());
}

void BoardSelectionForm::remove()
{
    View::remove();
    bsf = nullptr;
    bn->patch();
}

void init_top_header() { bn.reset(new BoardNavigation()); }
