#include "../brunhild/init.hh"
#include "../brunhild/mutations.hh"
#include "local_storage.hh"
#include "state.hh"
#include <emscripten.h>

int main()
{
    // TODO: This should be read from a concurrent server fetch
    char* conf = (char*)EM_ASM_INT_V({
        var s = JSON.stringify(window.config);
        var len = s.length + 1;
        var buf = Module._malloc(len);
        stringToUTF8(s, buf, len);
        return buf;
    });
    config = new Config(string(conf));
    delete[] conf;

    brunhild::set_inner_html("threads", config->default_css);
    brunhild::init();
    return 0;
}
