#ifndef CGO_VIDEO_H
#define CGO_VIDEO_H

#include <libavformat/avformat.h>

int extract_video_image(AVFrame **frame, AVFormatContext *ctx);

#endif
