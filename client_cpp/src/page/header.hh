#pragma once

#include "../../brunhild/view.hh"
#include "../form.hh"

// Beard navigation controls in the top banner
class BoardNavigation : public brunhild::VirtualView {
public:
    BoardNavigation();
    brunhild::Node render();

protected:
    void init();

private:
    // Renders a link to a board
    void board_link(
        std::ostringstream& s, const std::string& board, bool catalog);
};

// Beard navigation controls in the top banner
inline BoardNavigation board_navigation_view;

// From for selecting boards in BoardNavigation and more general board
// navigation
class BoardSelectionForm : public Form {
public:
    BoardSelectionForm();
    void remove();

protected:
    brunhild::Node render_inputs();
    brunhild::Node render_footer();
    brunhild::Node render_controls();

private:
    std::string filter;
};
