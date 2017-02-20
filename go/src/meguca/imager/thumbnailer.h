#ifndef CGO_THUMBNAILER_H
#define CGO_THUMBNAILER_H

#include "util.h"
#include <magick/api.h>
#include <stdbool.h>

struct Thumbnail {
	bool isPNG;
	struct Buffer img;
};

struct Options {
	uint8_t JPEGCompression;
	unsigned long maxSrcWidth, maxSrcHeight;
};

int thumbnail(struct Buffer *src,
			  struct Thumbnail *thumb,
			  const struct Options opts,
			  ExceptionInfo *ex);
static int writeThumb(Image *img,
					  struct Thumbnail *thumb,
					  const struct Options opts,
					  ExceptionInfo *ex);
static int
hasTransparency(const Image const *img, bool *needPNG, ExceptionInfo *ex);

#endif
