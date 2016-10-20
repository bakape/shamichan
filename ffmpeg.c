#include "ffmpeg.h"

int create_context(AVFormatContext **ctx, const int bufSize, const int flags)
{
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

int codec_context(AVCodecContext **codecCtx, AVFormatContext *ctx,
		  const enum AVMediaType type)
{
	AVCodec *codec = NULL;
	int strm = av_find_best_stream(ctx, type, -1, -1, &codec, 0);
	if (strm < 0) {
		return strm;
	}

	*codecCtx = ctx->streams[strm]->codec;
	int err = avcodec_open2(*codecCtx, codec, NULL);
	if (err < 0) {
		avcodec_free_context(codecCtx);
		return err;
	}

	return 0;
}

char *codec_name(AVFormatContext *ctx, const enum AVMediaType type,
		 const bool detailed)
{
	AVCodecContext *codecCtx = NULL;
	int err = codec_context(&codecCtx, ctx, type);
	if (err < 0) {
		return NULL;
	}

	const AVCodec *codec = codecCtx->codec;
	const char *src = detailed ? codec->long_name : codec->name;
	char *ret = malloc(strlen(src));
	strcpy(ret, src);
	avcodec_free_context(&codecCtx);

	return ret;
}

char *format_error(const int code)
{
	char *buf = malloc(1024);
	av_strerror(code, buf, 1024);
	return buf;
}
