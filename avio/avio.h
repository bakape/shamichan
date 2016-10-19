#ifndef CGO_AVIO_H
#define CGO_AVIO_H

#include <libavformat/avformat.h>
#include <stdbool.h>

extern int readCallBack(void *, uint8_t *, int);
extern int writeCallBack(void *, uint8_t *, int);
extern int64_t seekCallBack(void *, int64_t, int);

extern const int canRead;
extern const int canWrite;
extern const int canSeek;

AVFormatContext *format_context(AVFormatContext *ctx, const int bufSize,
				const int flags);
void destroy(AVFormatContext *ctx);
AVCodecContext *codec_context(AVFormatContext *ctx, enum AVMediaType type);
char *codec_name(AVFormatContext *ctx, enum AVMediaType type, bool detailed);

#endif
