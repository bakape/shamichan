#include "avio.h"
#include <errno.h>

AVFormatContext *format_context(AVFormatContext *ctx, const int bufSize,
				const int flags)
{
	errno = 0;

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
	ctx->pb = avio_alloc_context(buf, bufSize, 0, ctx, read, write, seek);

	int err = avformat_open_input(&ctx, NULL, NULL, NULL);
	if (err < 0) {
		errno = err;
		return NULL;
	}
	err = avformat_find_stream_info(ctx, NULL);
	if (err < 0) {
		errno = err;
		return NULL;
	}

	return ctx;
}

void destroy(AVFormatContext *ctx)
{
	av_free(ctx->pb->buffer);
	ctx->pb->buffer = NULL;
	av_free(ctx->pb);
	avformat_close_input(&ctx);
}

AVCodecContext *codec_context(AVFormatContext *ctx, const enum AVMediaType type)
{
	errno = 0;
	AVCodec *codec;
	if (av_find_best_stream(ctx, type, -1, -1, &codec, 0) < 0) {
		return NULL;
	}

	AVCodecContext *codecCtx = avcodec_alloc_context3(codec);
	int err = avcodec_open2(codecCtx, codec, NULL);
	if (err < 0) {
		errno = err;
		avcodec_free_context(&codecCtx);
		return NULL;
	}

	return codecCtx;
}

char *codec_name(AVFormatContext *ctx, const enum AVMediaType type,
		 const bool detailed)
{
	AVCodecContext *codecCtx = codec_context(ctx, type);
	if (codecCtx == NULL) {
		return NULL;
	}

	const AVCodec *codec = codecCtx->codec;
	const char *src = detailed ? codec->long_name : codec->name;
	char *ret = malloc(strlen(src));
	strcpy(ret, src);
	avcodec_free_context(&codecCtx);

	return ret;
}
