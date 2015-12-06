var config = module.exports = {
	IMAGE_FILESIZE_MAX: 1024 * 1024 * 3,
	IMAGE_WIDTH_MAX: 6000,
	IMAGE_HEIGHT_MAX: 6000,
	IMAGE_PIXELS_MAX: 4500*4500,
	MEDIA_DIRS: {
		src: 'www/src',
		thumb: 'www/thumb',
		mid: 'www/mid',
		tmp: 'imager/tmp'
	},
/*
 If using an external web server, set this to the served address of the www
 directory. Trailing slash required.
 */
	MEDIA_URL: '../',
// Set to separate upload address, if needed. Otherwise null
	UPLOAD_URL: null,

/*
 This should be the same as location.origin in your browser's javascript console
 */
	MAIN_SERVER_ORIGIN: 'http://localhost:8000',

/*
 Image duplicate detection threshold. Integer [0 - 256]. Higher is more
 agressive
 */
	DUPLICATE_THRESHOLD: 26,
/*
 * Thumbnail configuration for OP and regular thumbnails. Changing these will
 * cause existing images to have odd aspect ratios. It is recommended for THUMB
 * to be twice as big as PINKY.
 */
	PINKY_QUALITY: 50,
	PINKY_DIMENSIONS: [125, 125],
	THUMB_QUALITY: 50,
	THUMB_DIMENSIONS: [250, 250],
// Additional inbetween thumbnail quality setting. Served as "sharp"
	EXTRA_MID_THUMBNAILS: true,
// PNG thumbnails for PNG images. This enables thumbnail transparency.
	PNG_THUMBS: false,
// pngquant quality setting. Consult the manpages for more details
	PNG_THUMB_QUALITY: '0-10',
// Allow WebM video upload
	WEBM: false,
// Allow upload of WebM video with sound
	WEBM_AUDIO: false,
// MP3 upload
	MP3: false,
// Enable SVG upload
	SVG: false,
// Enable PDF upload
	PDF: false,

/*
 this indicates which spoiler images may be selected by posters.
 each number or ID corresponds to a set of images in ./www/spoil
 (named spoilX.png, spoilerX.png and spoilersX.png)
 */
	SPOILER_IMAGES: [1, 2, 3],

	IMAGE_HATS: false
};
