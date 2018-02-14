#include "../../brunhild/events.hh"
#include "image.hh"
#include <emscripten.h>

using brunhild::register_handler;

void init_posts()
{
    register_handler(
        "click", &handle_image_click, "figure img, figure video, figure a");
    register_handler("click", &toggle_hidden_thumbnail, ".image-toggle");

    // TODO: Remove this and the causing CSS, once transitioned to C++ client
    EM_ASM({
        var el = document.createElement('style');
        el.innerHTML = '.hash-link {display: unset;}';
        document.head.appendChild(el);
    });
}
