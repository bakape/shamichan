#include "../brunhild/init.hh"
#include "../brunhild/mutations.hh"
#include "local_storage.hh"
#include "posts/commands.hh"
#include "posts/init.hh"
#include "state.hh"
#include "util.hh"

int main()
{
    try {
        brunhild::before_flush = &rerender_syncwatches;
        brunhild::init();
        load_state();
        init_posts();
    } catch (const std::exception& ex) {
        console::error(ex.what());
    }
    return 0;
}
