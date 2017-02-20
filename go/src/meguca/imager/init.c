#include "init.h"
#include <magick/api.h>
#include <signal.h>
#include <stdio.h>
#include <string.h>

void magickInit()
{
	InitializeMagick(NULL);

#if defined(SIGCHLD)
	fixSignal(SIGCHLD);
#endif
#if defined(SIGHUP)
	fixSignal(SIGHUP);
#endif
#if defined(SIGINT)
	fixSignal(SIGINT);
#endif
#if defined(SIGQUIT)
	fixSignal(SIGQUIT);
#endif
#if defined(SIGABRT)
	fixSignal(SIGABRT);
#endif
#if defined(SIGFPE)
	fixSignal(SIGFPE);
#endif
#if defined(SIGTERM)
	fixSignal(SIGTERM);
#endif
#if defined(SIGBUS)
	fixSignal(SIGBUS);
#endif
#if defined(SIGSEGV)
	fixSignal(SIGSEGV);
#endif
#if defined(SIGXCPU)
	fixSignal(SIGXCPU);
#endif
#if defined(SIGXFSZ)
	fixSignal(SIGXFSZ);
#endif
}

// Add the SA_ONSTACK flag to a listened on signal to play nice with the Go
// runtime
static void fixSignal(int signum)
{
	struct sigaction st;

	if (sigaction(signum, NULL, &st) < 0) {
		return;
	}

	st.sa_flags |= SA_ONSTACK;
	sigaction(signum, &st, NULL);
}
