#include "ffmpeg.h"
#include <errno.h>

int extract_video_image(AVFrame **frame, AVFormatContext *ctx)
{
	AVCodec *codec = NULL;
	int strm =
	    av_find_best_stream(ctx, AVMEDIA_TYPE_VIDEO, -1, -1, &codec, 0);
	if (strm < 0) {
		return strm;
	}

	AVCodecContext *codecCtx = ctx->streams[strm]->codec;
	int err = avcodec_open2(codecCtx, codec, NULL);
	if (err < 0) {
		return err;
	}

	for (;;) {
		AVPacket pkt;
		int err = av_read_frame(ctx, &pkt);
		if (err < 0) {
			return err;
		}

		if (pkt.stream_index == strm) {
			int got = 0;
			*frame = av_frame_alloc();
			int err =
			    avcodec_decode_video2(codecCtx, *frame, &got, &pkt);
			av_free_packet(&pkt);
			if (err < 0) {
				av_frame_free(frame);
				return err;
			}

			if (got) {
				return 0;
			}
			av_frame_free(frame);
		}
	}
}
