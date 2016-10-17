#include "avio.h"

AVFormatContext *create_context(AVFormatContext *ctx)
{
	char errstringbuf[1024];
	int err = avformat_open_input(&ctx, NULL, NULL, NULL);
	if (err < 0) {
		av_strerror(err, errstringbuf, 1024);
		fprintf(stderr, "%s\n", errstringbuf);
		return NULL;
	}
	err = avformat_find_stream_info(ctx, NULL);
	if (err < 0) {
		av_strerror(err, errstringbuf, 1024);
		fprintf(stderr, "%s\n", errstringbuf);
		return NULL;
	}

	return ctx;
}
