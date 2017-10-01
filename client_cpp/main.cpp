#include "brunhild/init.hpp"
#include "brunhild/mutations.hpp"

int main()
{
    brunhild::init();
    brunhild::set_inner_html("threads", "Hello World!");
    return 0;
}
