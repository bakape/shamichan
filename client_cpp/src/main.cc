#include "../brunhild/init.hh"
#include "local_storage.hh"
#include "state.hh"

int main()
{
    load_state();
    brunhild::init();
    return 0;
}
