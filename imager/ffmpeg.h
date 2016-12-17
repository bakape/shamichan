#ifndef CGO_FFMPEG_H
#define CGO_FFMPEG_H

#include <libavformat/avformat.h>
#include <pthread.h>

extern int readCallBack(void *, uint8_t *, int);
extern int64_t seekCallBack(void *, int64_t, int);

int create_context(AVFormatContext **ctx);
void destroy(AVFormatContext *ctx);
int codec_context(AVCodecContext **avcc,
				  int *stream,
				  AVFormatContext *avfc,
				  const enum AVMediaType type);
char *format_error(const int code);

#endif
