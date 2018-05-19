#include "../../brunhild/events.hh"
#include "image.hh"
#include <emscripten.h>

using brunhild::register_handler;

void init_posts()
{
    register_handler(
        "click", &handle_image_click, "figure img, figure video, figure a");
    register_handler("click", &toggle_hidden_thumbnail, ".image-toggle");
}
