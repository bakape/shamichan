#include "../../brunhild/mutations.hh"
#include "../state.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <string>

void render_page()
{
    std::string buf;
    for (auto id : post_ids->mine) {
        buf += std::to_string(id);
        buf += ",";
    }
    EM_ASM_INT({ console.log(Pointer_stringify($0)); }, buf.c_str());
}
