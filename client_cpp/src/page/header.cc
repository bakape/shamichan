#include "header.hh"
#include "../../brunhild/mutations.hh"
#include "../lang.hh"
#include "../local_storage.hh"
#include "../state.hh"
#include "page.hh"
#include <algorithm>
#include <memory>
#include <set>

using brunhild::Node;

// Sort /all/ in front
struct board_sorter {
    bool operator()(const std::string& a, const std::string& b) const
    {
        if (a == "all") {
            return true;
        } else if (b == "all") {
            return true;
        }
        return a < b;
    }
};

static std::set<std::string, board_sorter> selected_boards;

// Board selection from instance
static std::unique_ptr<BoardSelectionForm> bsf;

// Returns, if board links should point to catalog pages
static bool point_to_catalog()
{
    const auto s = local_storage_get("pointToCatalog");
    if (!s) {
        return false;
    }
    return *s == "true";
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
    if (!selected_boards.size()) {
        selected_boards.insert("all");
    }
}

BoardNavigation::BoardNavigation()
    : VirtualView("board-navigation")
{
}

void BoardNavigation::init()
{
    read_selected();
    VirtualView::init();
    on("click", ".board-selection", [this](auto& _) {
        if (bsf) {
            bsf->remove();
        } else {
            bsf.reset(new BoardSelectionForm());
        }
        patch();
    });
}

Node BoardNavigation::render()
{
    std::ostringstream s;
    const bool catalog = point_to_catalog();
    s << '[';
    bool first = true;
    for (auto& b : selected_boards) {
        if (first) {
            first = false;
        } else {
            s << " / ";
        }
        s << "<a href=\"../" << b << '/';
        if (catalog) {
            s << "catalog";
        }
        s << "\">" << b << "</a>";
    }
    s << "] [<a class=\"board-selection bold mono\">" << (bsf ? "-" : "+")
      << "</a>]";
    return { "nav", { { "id", "board-navigation" } }, s.str() };
}

BoardSelectionForm::BoardSelectionForm()
    : Form(true)
{
    // Need to reduce any chance conflicts between multiple tabs
    read_selected();
    board_navigation_view.patch();

    on("input", "input[name=search]", [this](auto& event) {
        filter = to_lower(event["target"]["value"].template as<std::string>());
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

        board_navigation_view.patch();
    });

    brunhild::prepend("modal-overlay", html());
}

void BoardSelectionForm::remove()
{
    View::remove();
    bsf = nullptr;
    board_navigation_view.patch();
}

Node BoardSelectionForm::render_inputs()
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

Node BoardSelectionForm::render_footer()
{
    brunhild::Children ch;
    ch.reserve(boards.size());
    const bool to_catalog = point_to_catalog();
    std::ostringstream s;
    for (auto & [ board, title ] : boards) {
        s.str("");
        s << '/' << board << '/';
        if (to_catalog) {
            s << "catalog";
        }

        brunhild::Attrs attrs = { { "type", "checkbox" }, { "name", board } };
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

Node BoardSelectionForm::render_controls()
{
    brunhild::Attrs attrs
        = { { "type", "checkbox" }, { "name", "pointToCatalog" } };
    if (point_to_catalog()) {
        attrs["checked"] = "";
    }
    auto n = Form::render_controls();
    n.children.push_back({
        "label", {}, { { "input", attrs, lang.ui.at("pointToCatalog") } },
    });
    return n;
}

brunhild::Attrs BoardSelectionForm::attrs()
{
    return { { "class", "modal glass" }, { "style", "margin-left: .5em;" } };
}
