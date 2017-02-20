#ifndef CGO_THUMBNAILER_INIT_H
#define CGO_THUMBNAILER_INIT_H

void magickInit();
static void fixSignal(int signum);

#ifndef SA_ONSTACK
#define SA_ONSTACK 0x08000000
#endif

#endif
