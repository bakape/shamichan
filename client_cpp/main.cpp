#include "brunhild/init.hpp"
#include "brunhild/mutations.hpp"
#include <emscripten.h>

int main()
{
    init();

    set_inner_html("threads", "Hello World!");
    return 0;
}
