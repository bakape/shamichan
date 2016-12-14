#ifndef CGO_THUMBNAILER_H
#define CGO_THUMBNAILER_H

#include <magick/api.h>
#include <stdbool.h>
#include <stdint.h>

struct Thumbnail {
	void *buf;
	size_t size;
	unsigned long srcWidth, srcHeight, width, height;
};

struct Options {
	uint8_t JPEGCompression;
	unsigned long maxSrcWidth, maxSrcHeight;
};

int thumbnail(const void *src, const size_t size, const struct Options opts,
	      struct Thumbnail *thumb, ExceptionInfo *ex);
static void writeThumb(Image *img, struct Thumbnail *thumb,
		       const struct Options opts, ExceptionInfo *ex);

#endif
