#pragma once

#include "../brunhild/view.hh"
#include <emscripten/val.h>

// Generic input form view with optional captcha support
// TODO: Captcha support
class Form : public brunhild::VirtualView {
public:
    // Render form with optional specificied root node attributes
    // no_buttons: no Cancel or Submit buttons are rendered
    Form(bool no_buttons = false);

protected:
    virtual void init();

    // Return root element attributes
    virtual brunhild::Attrs attrs();

    // Handles sumbit event
    virtual void on_submit(emscripten::val&){};

    // Render submit and cancel buttons
    virtual brunhild::Node render_controls();

    // Render form input elements
    virtual brunhild::Node render_inputs() = 0;

    // Render any elements into the footer
    virtual brunhild::Node render_footer();

    brunhild::Node render();

    // Query all form input elements
    std::vector<emscripten::val> get_inputs();

private:
    const bool no_buttons;
};
