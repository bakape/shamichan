#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/inotify.h>
#include <unistd.h>

static const char *builder_pid = ".builder.pid", *server_pid = ".server.pid";

static void add_watch(const char *, void (*)(void));
static void kill_existing(const char *);

static void build_client(void) {
	if (!fork())
		execlp("make", "make", "www/js/client.js", NULL);
}

static void restart_server(void) {
	int pid;
	FILE *f;
	kill_existing(server_pid);
	if ((pid = fork()) < 0) {
		perror("fork");
	}
	else if (pid) {
		f = fopen(server_pid, "w");
		if (f) {
			fprintf(f, "%d\n", pid);
			fclose(f);
		}
	}
	else {
		execlp("node", "node", "server.js", NULL);
	}
}

static void client_and_server(void) {
	build_client();
	restart_server();
}

static void setup_watches(void) {
	add_watch("client.js", &build_client);
	add_watch("config.js", &build_client);
	add_watch("server.js", &restart_server);
	add_watch("common.js", &client_and_server);
}

static void kill_existing(const char *pid_file) {
	int pid;
	FILE *f = fopen(pid_file, "r");
	if (f) {
		if (fscanf(f, "%d", &pid) == 1)
			kill(pid, SIGTERM);
		fclose(f);
	}
}

#define MAX_WATCHES 4
static struct { int wd; void (*func)(void); } watches[MAX_WATCHES];
static int num_watches, inotify_fd;

static void monitor_files(void) {
	struct inotify_event event;
	int i;
	if (read(inotify_fd, &event, sizeof event) != sizeof event) {
		fprintf(stderr, "Monitor failure\n");
		exit(-1);
	}
	for (i = 0; i < MAX_WATCHES; i++) {
		if (event.wd == watches[i].wd) {
			(*watches[i].func)();
			break;
		}
	}
}

static void add_watch(const char *filename, void (*f)(void)) {
	int w;
	if (num_watches >= MAX_WATCHES) {
		fprintf(stderr, "No slots available for more watches.\n");
		return;
	}
	w = inotify_add_watch(inotify_fd, filename, IN_MODIFY);
	if (w < 0) {
		perror(filename);
		return;
	}
	watches[num_watches].wd = w;
	watches[num_watches].func = f;
	num_watches++;
}

static void daemonize(void) {
	FILE *f;
	int pid = fork();
	if (pid < 0) {
		perror("fork");
		exit(-1);
	}
	else if (pid) {
		f = fopen(builder_pid, "w");
		if (f) {
			fprintf(f, "%d\n", pid);
			fclose(f);
		}
		printf("Forked monitor.\n");
		exit(0);
	}
}

int main(void) {
	FILE *f;
	struct sigaction act;

	kill_existing(builder_pid);

	inotify_fd = inotify_init();
	if (!inotify_fd) {
		perror("inotify_init");
		return -1;
	}
	setup_watches();
	restart_server();

	daemonize();

	while (1)
		monitor_files();
}
