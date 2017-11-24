#include "../brunhild/init.hh"
#include "../brunhild/mutations.hh"
#include "local_storage.hh"
#include "posts/commands.hh"
#include "state.hh"
#include "util.hh"

int main()
{
    try {
        load_state();
        brunhild::before_flush = &rerender_syncwatches;
        brunhild::init();
    } catch (const std::exception& ex) {
        console::error(ex.what());
    }
    return 0;
}
