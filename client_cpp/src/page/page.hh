#pragma once

#include "../../brunhild/view.hh"
#include "../lang.hh"
#include <string>

// Render the page, after all the internal state has been loaded
void render_page();

// Format and escape a page title
std::string format_title(const std::string& board, const std::string& text);

// Set's the title of the page. Requires escaping of the string.
void set_title(std::string);

// Push board-specific hover-revealed information elements, if any, to ch.
void push_board_hover_info(brunhild::Children& ch);

// Top banner with a board-specific images
class ImageBanner : public brunhild::VirtualView {
    brunhild::Node render();

    // TODO: Change on click
};

// Contains the post-related portion of the page
class PageView : public brunhild::CompositeView<> {
public:
    PageView()
        : CompositeView<>("section", "threads")
    {
    }

protected:
    std::vector<brunhild::View*> get_list() final;

    // Returns the top controls of the page
    virtual std::vector<brunhild::View*> top_controls() = 0;

    //  Returns the view containing the threads and posts
    virtual brunhild::View* thread_container() = 0;

    // Returns the bottom controls of the page
    virtual std::vector<brunhild::View*> bottom_controls() = 0;

private:
    // Row of various <aside> elements on page top and bottom
    // bottom: row is at page bottom
    class AsideRow : public brunhild::CompositeView<> {
    public:
        AsideRow(std::vector<brunhild::View*> list, bool bottom);

    protected:
        std::vector<brunhild::View*> get_list() { return list; }
        brunhild::Attrs attrs();

    private:
        std::vector<brunhild::View*> list;
    };
};

// Simple static button
class Button : public brunhild::NodeView {
public:
    // Creates button with optional url on link
    Button(std::string text, std::optional<std::string> href = {});
};

// Static control that reveals text on hover
class HoverTooltip : public brunhild::NodeView {
public:
    HoverTooltip(std::string label_id, std::string text);
};
