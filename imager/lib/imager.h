#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

bool is_tar_gz(const uint8_t *buf, size_t size);
bool is_tar_xz(const uint8_t *buf, size_t size);
