#include "ffmpeg.h"

const int bufSize = 1 << 12;

// Initialize am AVFormatContext with the buffered file
int create_context(AVFormatContext **ctx)
{
	unsigned char *buf = malloc(bufSize);
	AVFormatContext *c = *ctx;

	c->pb = avio_alloc_context(buf, bufSize, 0, c, readCallBack, NULL,
				   seekCallBack);
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

// Create a AVCodecContext of the desired media type
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

// Format ffmpeg error code to string message
char *format_error(const int code)
{
	char *buf = malloc(1024);
	av_strerror(code, buf, 1024);
	return buf;
}
