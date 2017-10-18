#include "../brunhild/init.hh"
#include "local_storage.hh"
#include "state.hh"
#include "util.hh"

int main()
{
    try {
        load_state();
        brunhild::init();
    } catch (const std::exception& ex) {
        console_log(ex.what());
    }
    return 0;
}
