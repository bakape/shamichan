#include "video.h"
#include <libavutil/imgutils.h>

const size_t frameBufferSize = 1 << 17;

int extract_video_image(struct Buffer *img,
						AVFormatContext *avfc,
						AVCodecContext *avcc,
						const int stream)
{
	AVFrame *frame;
	AVPacket pkt;
	int err, got;

	for (;;) {
		err = av_read_frame(avfc, &pkt);
		if (err < 0) {
			return err;
		}

		if (pkt.stream_index == stream) {
			got = 0;
			frame = av_frame_alloc();
			err = avcodec_decode_video2(avcc, frame, &got, &pkt);
			av_packet_unref(&pkt);
			if (err < 0) {
				av_frame_free(&frame);
				return err;
			}

			if (got) {
				err = encode_frame(img, frame);
				av_frame_free(&frame);
				return err;
			}
			av_frame_free(&frame);
		}
	}
}

// Encode frame to YUV420 image
static int encode_frame(struct Buffer *img, const AVFrame const *frame)
{
	int ret;

	img->size = (size_t)av_image_get_buffer_size(
		frame->format, frame->width, frame->height, 1);
	img->width = (unsigned long)frame->width;
	img->height = (unsigned long)frame->height;
	img->data = malloc(img->size);

	ret = av_image_copy_to_buffer(img->data,
								  img->size,
								  (const unsigned char *const *)frame->data,
								  frame->linesize,
								  frame->format,
								  frame->width,
								  frame->height,
								  1);
	if (ret < 0) {
		return ret;
	}
	img->size = ret;
	return 0;
}
