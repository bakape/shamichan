#ifndef CGO_AVIO_H
#define CGO_AVIO_H

#include <libavformat/avformat.h>

extern int readCallBack(void *, uint8_t *, int);
extern int writeCallBack(void *, uint8_t *, int);
extern int64_t seekCallBack(void *, int64_t, int);

AVFormatContext *create_context(AVFormatContext *ctx);
void destroy(AVFormatContext *ctx);

#endif
