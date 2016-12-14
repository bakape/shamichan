#ifndef CGO_FFMPEG_AUDIO_H
#define CGO_FFMPEG_AUDIO_H

#include "ffmpeg.h"

AVPacket retrieve_cover_art(AVFormatContext *ctx);
int find_cover_art(AVFormatContext *ctx);

#endif
