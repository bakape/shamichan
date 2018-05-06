#pragma once

#include "../brunhild/view.hh"
#include "lang.hh"
#include "state.hh"
#include <emscripten/val.h>
#include <string>
#include <unordered_map>

// Generic input form view with optional captcha support
// TODO: Captcha support
class Form : public brunhild::View {
public:
    // Render form with optional specificied root node attributes
    // no_buttons: no Cancel or Submit buttons are rendered
    Form(brunhild::Attrs attrs = {}, bool no_buttons = false)
        : attrs(attrs)
        , no_buttons(no_buttons)
    {
    }

    // Query all form input elements
    std::vector<emscripten::val> get_inputs();

protected:
    const brunhild::Attrs attrs;

    virtual void init();

    // Handles sumbit event
    virtual void on_submit(emscripten::val&){};

    // Render any elements after the submit and cancel buttons
    virtual brunhild::Children render_after_controls() { return {}; }

    // Render form input elements
    virtual brunhild::Node render_inputs() = 0;

    // Render any elements into the footer
    virtual brunhild::Node render_footer() { return {}; }

private:
    const bool no_buttons;

    brunhild::Node render();
};
