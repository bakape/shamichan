#include "../options/main.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <string>

void render_page()
{
    std::string buf = options->theme;
    EM_ASM_INT({ console.log(Pointer_stringify($0)); }, buf.c_str());
}

EMSCRIPTEN_BINDINGS(module_page)
{
    emscripten::function("render_page", &render_page);
}
