#include "../lang.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <string>

void render_page()
{
    EM_ASM_INT(
        { console.log(Pointer_stringify($0)); }, lang->posts["you"].c_str());
}

EMSCRIPTEN_BINDINGS(module_page)
{
    emscripten::function("render_page", &render_page);
}
