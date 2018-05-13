#include "form.hh"
#include <emscripten/val.h>

void Form::init()
{
    VirtualView::init();
    if (!no_buttons) {
        on("click", "input[name=cancel]", [this](auto& _) { remove(); });
        on("submit", "", [this](auto& event) { on_submit(event); });
    }
}

brunhild::Node Form::render()
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

    const auto after = render_after_controls();
    controls.children.insert(
        controls.children.end(), after.begin(), after.end());

    return { "form", attrs, { render_inputs(), controls, render_footer() } };
}

std::vector<emscripten::val> Form::get_inputs()
{
    using namespace emscripten;

    return vecFromJSArray<val>(val::global("document")
                                   .call<val>("getElementById", id)
                                   .call<val>("querySelectorAll",
                                       std::string("input,textarea,select")));
}
