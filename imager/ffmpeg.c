#include "ffmpeg.h"

int create_context(AVFormatContext **ctx, const int bufSize, const int flags)
{
	// Pick which custom I/O callbacks to assign to AVIOContext
	int (*read)(void *, uint8_t *, int);
	int (*write)(void *, uint8_t *, int);
	int64_t (*seek)(void *, int64_t, int);
	if (flags & canRead) {
		read = readCallBack;
	}
	if (flags & canWrite) {
		write = writeCallBack;
	}
	if (flags & canSeek) {
		seek = seekCallBack;
	}

	unsigned char *buf = malloc((size_t)(bufSize));
	AVFormatContext *c = *ctx;
	c->pb = avio_alloc_context(buf, bufSize, 0, c, read, write, seek);
	c->flags |= AVFMT_FLAG_CUSTOM_IO;

	int err = avformat_open_input(ctx, NULL, NULL, NULL);
	if (err < 0) {
		return err;
	}
	err = avformat_find_stream_info(*ctx, NULL);
	if (err < 0) {
		return err;
	}

	return 0;
}

void destroy(AVFormatContext *ctx)
{
	av_free(ctx->pb->buffer);
	ctx->pb->buffer = NULL;
	av_free(ctx->pb);
	av_free(ctx);
}

int codec_context(AVCodecContext **avcc, int *stream, AVFormatContext *avfc,
		  const enum AVMediaType type)
{
	AVCodec *codec = NULL;
	*stream = av_find_best_stream(avfc, type, -1, -1, &codec, 0);
	if (*stream < 0) {
		return *stream;
	}

	*avcc = avfc->streams[*stream]->codec;
	int err = avcodec_open2(*avcc, codec, NULL);
	if (err < 0) {
		avcodec_free_context(avcc);
		return err;
	}

	return 0;
}

char *format_error(const int code)
{
	char *buf = malloc(1024);
	av_strerror(code, buf, 1024);
	return buf;
}
