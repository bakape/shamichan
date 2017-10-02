#include "mutations.hh"
#include <emscripten.h>

namespace brunhild {
void init()
{
    // TODO: Set up default event listeners, including the ones buffering
    // input element status.
             emscripten_set_main_loop(flush, 0, 0);
}
}
