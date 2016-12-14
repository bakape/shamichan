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

int thumbnail(const void *src, const size_t size, const struct Options opts,
	      struct Thumbnail *thumb, ExceptionInfo *ex);
static void writeThumb(Image *img, struct Thumbnail *thumb,
		       const struct Options opts, ExceptionInfo *ex);

#endif
