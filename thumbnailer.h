#ifndef CGO_THUMBNAILER_H
#define CGO_THUMBNAILER_H

#include "stdbool.h"
#include <magick/api.h>

struct Thumbnail {
	void *buf;
	size_t size;
	unsigned long width, height;
};

struct Options {
	int outputType;
	unsigned long width, height, JPEGCompression;
};

int thumbnail(const void *, const size_t, const struct Options,
	      struct Thumbnail *, ExceptionInfo *);

#endif
