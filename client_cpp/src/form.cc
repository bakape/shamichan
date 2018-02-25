#include "form.hh"

void Form::init()
{
    View::init();
    on("click", "input[name=cancel]", [this](auto& _) { remove(); });
    on("submit", "", [this](auto& event) { on_submit(event); });
}

brunhild::Node Form::render()
{
    brunhild::Node controls = {
        "span",
        { { "class", "flex" } },
        {
            {
                "input",
                {
                    { "type", "submit" },
                    { "value", lang.ui.at("submit") },
                },
            },
            {
                "input",
                {
                    { "type", "button" },
                    { "value", lang.ui.at("cancel") },
                    { "name", "cancel" },
                },
            },
        },
    };
    const auto after = render_after_controls();
    controls.children.insert(
        controls.children.end(), after.begin(), after.end());

    return {
        "form",
        attrs,
        { render_inputs(), controls, render_footer() },
    };
}
