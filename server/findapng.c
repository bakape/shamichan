#include <png.h>
#include <stdio.h>

#define FAIL(format, args...) do { fprintf(stderr, format "\n", ## args); \
		goto done; } while (0)

static int apng = 0;

static int read_chunk(png_structp png_ptr, png_unknown_chunkp chunk) {
	(void) png_ptr;
	if (strncmp((const char *) chunk, "acTL", 4) == 0)
		apng = 1;
	return 0;
}

int main(int argc, char *argv[]) {
	FILE *fp;
	const char *filename;
	unsigned char header[8];
	png_structp png_ptr = NULL;
	png_infop info_ptr = NULL;
	int result = 1;

	if (argc != 2) {
		fprintf(stderr, "Usage: %s <png>\n", argv[0]);
		return -1;
	}
	filename = argv[1];
	fp = fopen(filename, "rb");
	if (!fp) {
		perror(filename);
		return -1;
	}

	if (fread(header, 8, 1, fp) != 1)
		FAIL("%s: Couldn't read header.", filename);
	if (png_sig_cmp(header, 0, 8))
		FAIL("%s: Not a PNG.", filename);
	png_ptr = png_create_read_struct(PNG_LIBPNG_VER_STRING,
			NULL, NULL, NULL);
	if (!png_ptr)
		FAIL("Couldn't set up PNG reader.");
	info_ptr = png_create_info_struct(png_ptr);
	if (!info_ptr)
		FAIL("Couldn't set up PNG info reader.");

	if (setjmp(png_jmpbuf(png_ptr)))
		goto done;

	png_init_io(png_ptr, fp);
	png_set_sig_bytes(png_ptr, 8);
	png_set_read_user_chunk_fn(png_ptr, NULL, &read_chunk);
	png_set_keep_unknown_chunks(png_ptr, PNG_HANDLE_CHUNK_NEVER, NULL, 0);
	png_read_info(png_ptr, info_ptr);

	puts(apng ? "APNG" : "PNG");
	result = 0;
done:
	if (png_ptr)
		png_destroy_read_struct(&png_ptr, &info_ptr, NULL);
	fclose(fp);
	return result;
}
