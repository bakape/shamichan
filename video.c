#include "video.h"

int extract_video_image(AVFrame **frame, AVFormatContext *avfc,
			AVCodecContext *avcc, const int stream)
{
	for (;;) {
		AVPacket pkt;
		int err = av_read_frame(avfc, &pkt);
		if (err < 0) {
			return err;
		}

		if (pkt.stream_index == stream) {
			int got = 0;
			*frame = av_frame_alloc();
			err = avcodec_decode_video2(avcc, *frame, &got, &pkt);
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
