#include "brunhild/init.hh"
#include "brunhild/mutations.hh"

int main()
{
    brunhild::init();
    brunhild::set_inner_html("threads", "Hello World!");
    return 0;
}
