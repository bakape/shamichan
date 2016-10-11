#include <libavformat/avformat.h>

extern int readCallBack(void *, uint8_t *, int);
extern int writeCallBack(void *, uint8_t *, int);
extern int64_t seekCallBack(void *, int64_t, int);

AVFormatContext *create_context(AVFormatContext *ctx);
AVFrame *extract_video_image(AVFormatContext *ctx);
AVCodecContext *extract_video(AVFormatContext *ctx);
AVCodecContext *extract_audio(AVFormatContext *ctx);
