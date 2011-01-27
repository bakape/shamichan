#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/inotify.h>
#include <sys/wait.h>
#include <time.h>
#include <unistd.h>

static const char *builder_pid = ".builder.pid", *server_pid = ".server.pid";

static void add_watch(const char *, void (*)(void));
static void kill_existing(const char *);
static void read_version(char *);

static void build_client(void) {
	char buf[32] = "../www/js/client-";
	read_version(buf + strlen(buf));
	strcat(buf, ".js");
	if (!fork())
		execlp("make", "make", "-s", buf, NULL);
}

static void restart_server(void) {
	kill_existing(server_pid);
}

static void client_and_server(void) {
	build_client();
	restart_server();
}

static void setup_watches(void) {
	add_watch("client.js", &build_client);
	add_watch("config.js", &restart_server);
	add_watch("server.js", &restart_server);
	add_watch("tripcode.node", &restart_server);
	add_watch("index.html", &restart_server);
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

#define MAX_WATCHES 6
static struct { const char *name; void (*func)(void); time_t stamp; }
		watches[MAX_WATCHES];
static int num_watches, inotify_fd;

static void monitor_files(void) {
	struct inotify_event *event;
	char buf[128];
	int i, len, total;
	time_t now;
	if ((total = read(inotify_fd, buf, sizeof buf)) <= 0) {
		fprintf(stderr, "Monitor failure\n");
		exit(-1);
	}
	event = (struct inotify_event *) buf;
	while (total > 0) {
		now = time(NULL);
		for (i = 0; i < MAX_WATCHES; i++) {
			if (!strcmp(event->name, watches[i].name)
					&& now - 1 > watches[i].stamp) {
				(*watches[i].func)();
				watches[i].stamp = now;
				break;
			}
		}
		len = sizeof *event + event->len;
		event = (struct inotify_event *) ((char *)event + len);
		total -= len;
	}
}

static void add_watch(const char *filename, void (*f)(void)) {
	if (num_watches >= MAX_WATCHES) {
		fprintf(stderr, "No slots available for more watches.\n");
		return;
	}
	watches[num_watches].name = filename;
	watches[num_watches].func = f;
	watches[num_watches].stamp = time(NULL);
	num_watches++;
}

static void read_version(char *dest) {
	FILE *f = popen("node config.js --version", "r");
	if (!f || fscanf(f, "%10s", dest) != 1) {
		fprintf(stderr, "Couldn't read version.\n");
		exit(-1);
	}
	pclose(f);
}

static void server_process() {
	FILE *f;
	int pid;
	time_t start;

	do {
		start = time(NULL);
		/* turn main process into the server */
		pid = fork();
		if (pid < 0)
			break;
		else if (!pid) {
			printf("Running server.\n");
			execlp("node", "node", "server.js", NULL);
			perror("node server.js");
			break;
		}

		f = fopen(server_pid, "w");
		if (f) {
			fprintf(f, "%d\n", pid);
			fclose(f);
		}
		waitpid(pid, NULL, 0);
	} while (time(NULL) > start + 2);
}

int main(int argc, char *argv[]) {
	FILE *f;
	int pid;

	if (argc > 1 && !strcmp(argv[1], "--server")) {
		server_process();
		fprintf(stderr, "Server stopped.\n");
		return -1;
	}

	kill_existing(builder_pid);

	inotify_fd = inotify_init();
	if (!inotify_fd) {
		perror("inotify_init");
		return -1;
	}
	if (inotify_add_watch(inotify_fd, ".", IN_CREATE | IN_MODIFY) < 0) {
		perror(".");
		return -1;
	}
	setup_watches();
	client_and_server();

	pid = fork();
	if (pid < 0) {
		perror("fork");
		exit(-1);
	}
	else if (!pid) {
		while (1)
			monitor_files();
	}

	printf("Forked monitor.\n");
	f = fopen(builder_pid, "w");
	if (f) {
		fprintf(f, "%d\n", pid);
		fclose(f);
	}

	execlp(argv[0], argv[0], "--server", NULL);
}
