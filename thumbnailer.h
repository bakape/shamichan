#ifndef CGO_THUMBNAILER_H
#define CGO_THUMBNAILER_H

#include "stdbool.h"
#include <magick/api.h>

unsigned long maxX, maxY;

struct Thumbnail {
	void *buf;
	size_t size;
	unsigned long width;
	unsigned long height;
};

int thumbnail(const void *src, const size_t size, struct Thumbnail *thumb,
	      bool jpeg, ExceptionInfo *ex);

#endif
