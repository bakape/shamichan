#include "form.hh"
#include "lang.hh"
#include <emscripten/val.h>

Form::Form(bool no_buttons)
    : no_buttons(no_buttons)
{
}

void Form::init()
{
    VirtualView::init();
    if (!no_buttons) {
        on("click", "input[name=cancel]", [this](auto& _) { remove(); });
        on("submit", "", [this](auto& event) { on_submit(event); });
    }
}

brunhild::Attrs Form::attrs() { return {}; }

brunhild::Node Form::render()
{
    return { "form", attrs(),
        { render_inputs(), render_controls(), render_footer() } };
}

brunhild::Node Form::render_controls()
{
    brunhild::Node controls = { "span", { { "class", "flex" } } };
    if (!no_buttons) {
        controls.children = {
            {
                "input",
                {
                    { "type", "submit" }, { "value", lang.ui.at("submit") },
                },
            },
            {
                "input",
                {
                    { "type", "button" }, { "value", lang.ui.at("cancel") },
                    { "name", "cancel" },
                },
            },
        };
    }
    return controls;
}

brunhild::Node Form::render_footer() { return {}; }

std::vector<emscripten::val> Form::get_inputs()
{
    using namespace emscripten;

    return vecFromJSArray<val>(val::global("document")
                                   .call<val>("getElementById", id)
                                   .call<val>("querySelectorAll",
                                       std::string("input,textarea,select")));
}
