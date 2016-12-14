#include "string.h"
#include "thumbnailer.h"

int thumbnail(const void *src, const size_t size, const struct Options opts,
	      struct Thumbnail *thumb, ExceptionInfo *ex)
{
	ImageInfo *info = NULL;
	Image *img = NULL, *sampled = NULL, *scaled = NULL;
	double scale;

	// Read image
	info = CloneImageInfo(NULL);
	GetExceptionInfo(ex);
	img = BlobToImage(info, src, size, ex);
	if (img == NULL) {
		goto end;
	}

	// Image already fits thumbnail
	if (img->columns <= opts.width && img->rows <= opts.height) {
		thumb->width = img->columns;
		thumb->height = img->rows;
		writeThumb(img, thumb, opts, ex);
		goto end;
	}

	// Maintain aspect ratio
	if (img->columns >= img->rows) {
		scale = (double)(img->columns) / (double)(opts.width);
	} else {
		scale = (double)(img->rows) / (double)(opts.height);
	}
	thumb->width = (unsigned long)(img->columns / scale);
	thumb->height = (unsigned long)(img->rows / scale);

	// Subsample to twice the thumbnail size. A decent enough compromise
	// between quality and performance for images arround the thumbnail size
	// and much bigger ones.
	sampled = SampleImage(img, thumb->width * 2, thumb->height * 2, ex);
	if (sampled == NULL) {
		goto end;
	}

	// Scale to thumbnail size
	scaled = ResizeImage(sampled, thumb->width, thumb->height, CubicFilter,
			     1, ex);
	if (scaled == NULL) {
		goto end;
	}

	writeThumb(scaled, thumb, opts, ex);

end:
	if (img != NULL) {
		DestroyImage(img);
	}
	if (info != NULL) {
		DestroyImageInfo(info);
	}
	if (sampled != NULL) {
		DestroyImage(sampled);
	}
	if (scaled != NULL) {
		DestroyImage(scaled);
	}
	return thumb->buf == NULL;
}

static void writeThumb(Image *img, struct Thumbnail *thumb,
		       const struct Options opts, ExceptionInfo *ex)
{
	ImageInfo *info = CloneImageInfo(NULL);

	if (opts.outputType) {
		info->quality = opts.JPEGCompression;
		strcpy(info->magick, "JPEG");
		strcpy(img->magick, "JPEG");
	} else {
		// Will pass through libimagequant, so comression and filters
		// are pointeless
		info->quality = 0;
		strcpy(info->magick, "PNG");
		strcpy(img->magick, "PNG");
	}
	thumb->buf = ImageToBlob(info, img, &thumb->size, ex);

	DestroyImageInfo(info);
}
