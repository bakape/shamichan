#ifndef CGO_UTIL_H
#define CGO_UTIL_H

#include <stddef.h>
#include <stdint.h>

struct Buffer {
	uint8_t *data;
	size_t size;
	unsigned long width, height;
};

#endif
