#pragma once

#include "../brunhild/view.hh"
#include "lang.hh"
#include "state.hh"
#include <emscripten/val.h>
#include <string>

// Generic input form view with optional captcha support
// TODO: Captcha support
class Form : public brunhild::View {
public:
    // Render form with optional specificied root node attributes
    Form(brunhild::Attrs attrs = {})
        : attrs(attrs)
    {
    }

    void init();

    // Add special logic to exec on event removal
    virtual void remove() { View::remove(); }

protected:
    const brunhild::Attrs attrs;

    // Handles sumbit event
    virtual void on_submit(emscripten::val) = 0;

    // Render any elements after the submit and cancel buttons
    virtual brunhild::Children render_after_controls() { return {}; }

    // Render form input elements
    virtual brunhild::Node render_inputs() = 0;

    // Render any elements into the footer
    virtual brunhild::Node render_footer() { return {}; }

private:
    brunhild::Node render();
};
