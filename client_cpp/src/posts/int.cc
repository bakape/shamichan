#include "../../brunhild/events.hh"
#include "image.hh"
#include <emscripten.h>

using brunhild::register_handler;

void init_posts()
{
    register_handler(
        "click", &handle_image_click, "figure img, figure video, figure a");
    register_handler("click", &toggle_hidden_thumbnail, ".image-toggle");

    // Block all clicks on <a> from exhibiting browser default behavior, unless
    // the user intends to navigate to a new tab or open a browser menu
    EM_ASM({
        document.addEventListener("click", function(e) {
            if (e.which != 1 || e.ctrlKey) {
                return;
            }
            var t = e.target;
            switch (t.tagName) {
            case "A":
                if (t.getAttribute("target") == "_blank"
                    || t.getAttribute("download")) {
                    return;
                }
            case "IMG":
                e.preventDefault();
            }
        });
    });
}
