#include <stdio.h>
#include <stdlib.h>

int main(int argc, char *argv[]) {
	const char *filename;
	FILE *f;
	unsigned char buf[256], c, *p;
	unsigned int i, mean;
	int bit;

	if (argc != 2) {
		fprintf(stderr, "Usage: %s img.gray\n", argv[0]);
		return -1;
	}
	filename = argv[1];
	f = fopen(filename, "rb");
	if(!f) {
		perror(filename);
		return -1;
	}
	if (fread(buf, sizeof buf, 1, f) != 1) {
		perror(filename);
		fclose(f);
		return -1;
	}
	fclose(f);
	mean = 0;
	for (i = 0; i < sizeof buf; i++)
		mean += buf[i];
	mean /= sizeof buf;
	p = buf;
	for (i = 0; i < sizeof buf / 4; i++) {
		c = 0;
		for (bit = 3; bit >= 0; bit--)
			c |= (*p++ > mean) << bit;
		putchar("0123456789abcdef"[c]);
	}
	return 0;
}
