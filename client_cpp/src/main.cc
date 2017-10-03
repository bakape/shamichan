#include "../brunhild/init.hh"
#include "../brunhild/mutations.hh"
#include "local_storage.hh"

int main()
{
    brunhild::init();
    brunhild::set_inner_html("threads", local_storage_get("theme"));
    return 0;
}
