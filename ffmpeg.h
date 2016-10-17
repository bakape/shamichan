#ifndef CGO_VIDEO_H
#define CGO_VIDEO_H

#include <libavformat/avformat.h>

AVFrame *extract_video_image(AVFormatContext *ctx);
AVCodecContext *extract_video(AVFormatContext *ctx);
AVCodecContext *extract_audio(AVFormatContext *ctx);

#endif
