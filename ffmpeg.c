#include <libavcodec/avcodec.h>
#include <libavutil/frame.h>
#include <libavutil/avutil.h>
#include <libavformat/avformat.h>
#include <libavformat/avio.h>
#include <libswscale/swscale.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include "ffmpeg.h"

AVFormatContext * create_context(AVFormatContext *ctx)
{
	char errstringbuf[1024];
	int err = avformat_open_input(&ctx, NULL, NULL, NULL);
	if (err < 0) {
		av_strerror(err,errstringbuf,1024);
		fprintf(stderr,"%s\n",errstringbuf);
		return NULL;
	}
	err = avformat_find_stream_info(ctx,NULL);
	if (err < 0) {
		av_strerror(err,errstringbuf,1024);
		fprintf(stderr,"%s\n",errstringbuf);
		return NULL;
	}

	return ctx;
}

AVFrame * extract_video_image(AVFormatContext *ctx)
{
	char errstringbuf[1024];
	int err;
	AVCodec * codec = NULL;
	//Get video stream ID
	int strm = av_find_best_stream(ctx, AVMEDIA_TYPE_VIDEO, -1, -1, &codec, 0);

	AVCodecContext * codecCtx = ctx->streams[strm]->codec;
	//Open video decoder
	err = avcodec_open2(codecCtx, codec, NULL);
	if (err < 0) {
		av_strerror(err,errstringbuf,1024);
		fprintf(stderr,"%s\n",errstringbuf);
		return NULL;
	}


	for (;;)
	{
		AVPacket pkt;
		err = av_read_frame(ctx, &pkt);
		if (err < 0) {
			av_strerror(err,errstringbuf,1024);
			fprintf(stderr,"%s\n",errstringbuf);
			return NULL;
		}

		if (pkt.stream_index == strm)
		{
			int got = 0;
			AVFrame * frame = av_frame_alloc();
			err = avcodec_decode_video2(codecCtx, frame, &got, &pkt);
			if (err < 0) {
				av_strerror(err,errstringbuf,1024);
				fprintf(stderr,"%s\n",errstringbuf);
				return NULL;
			}

			if (got) {
				return frame;
			}
			av_frame_free(&frame);
		}
	}
}

AVCodecContext * extract_video(AVFormatContext *ctx)
{
	char errstringbuf[1024];
	int err;
	AVCodec * codec = NULL;
	int strm = av_find_best_stream(ctx, AVMEDIA_TYPE_VIDEO, -1, -1, &codec, 0);

	AVCodecContext * codecCtx = ctx->streams[strm]->codec;
	err = avcodec_open2(codecCtx, codec, NULL);
	if (err < 0) {
		av_strerror(err,errstringbuf,1024);
		fprintf(stderr,"%s\n",errstringbuf);
		return NULL;
	}
	return codecCtx;
}

AVCodecContext * extract_audio(AVFormatContext *ctx)
{
	char errstringbuf[1024];
	int err;
	AVCodec * codec = NULL;
	int strm = av_find_best_stream(ctx, AVMEDIA_TYPE_AUDIO, -1, -1, &codec, 0);
	if (strm < 0) {
		return NULL;
	}
	AVCodecContext * codecCtx = ctx->streams[strm]->codec;
	err = avcodec_open2(codecCtx, codec, NULL);
	if (err < 0) {
		av_strerror(err,errstringbuf,1024);
		fprintf(stderr,"%s\n",errstringbuf);
		return NULL;
	}
	return codecCtx;
}
