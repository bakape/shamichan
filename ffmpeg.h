#ifndef CGO_FFMPEG_H
#define CGO_FFMPEG_H

#include <libavformat/avformat.h>

extern int readCallBack(void *, uint8_t *, int);
extern int writeCallBack(void *, uint8_t *, int);
extern int64_t seekCallBack(void *, int64_t, int);

extern const int canRead;
extern const int canWrite;
extern const int canSeek;

int create_context(AVFormatContext **ctx, const int bufSize, const int flags);
void destroy(AVFormatContext *ctx);
int codec_context(AVCodecContext **avcc, int *stream, AVFormatContext *avfc,
		  const enum AVMediaType type);
char *format_error(const int code);

#endif
