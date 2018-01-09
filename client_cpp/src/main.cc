#include "../brunhild/init.hh"
#include "../brunhild/mutations.hh"
#include "local_storage.hh"
#include "posts/commands.hh"
#include "posts/init.hh"
#include "state.hh"

int main()
{
    brunhild::before_flush = &rerender_syncwatches;
    brunhild::init();
    load_state();
    init_posts();
    return 0;
}
