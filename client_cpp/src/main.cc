#include "../brunhild/init.hh"
#include "../brunhild/mutations.hh"
#include "local_storage.hh"
#include "state.hh"

int main()
{
    load_state();

    brunhild::set_inner_html(
        "threads", page->board + std::to_string(page->thread));
    brunhild::init();
    return 0;
}
