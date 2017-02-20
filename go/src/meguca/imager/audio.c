#include "audio.h"

// Extract embedded image
AVPacket retrieve_cover_art(AVFormatContext *ctx)
{
	const int i = find_cover_art(ctx);
	if (i != -1) {
		return ctx->streams[i]->attached_pic;
	}

	AVPacket err;
	return err;
}

// Find the first attached picture, if available
int find_cover_art(AVFormatContext *ctx)
{
	for (int i = 0; i < ctx->nb_streams; i++) {
		const int d = ctx->streams[i]->disposition;
		if (d & AV_DISPOSITION_ATTACHED_PIC) {
			return i;
		}
	}
	return -1;
}
