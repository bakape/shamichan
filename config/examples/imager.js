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

 If using Cloudflare and serving images through HTTPS, take note that Cloudflare
 only supports a limited range of cyphers. IQDB and saucenao at the moment of
 writing fail the SSL handshake, when attempting to download the thumbnail from
 the server. As a workaround the image seach URLs for these will be appended the
 '?ssl=off' query string. You must manually set a page rule with Cloudflare to
 not use SSL for URLs with this string. Also, enable chaching of query string
 resources.

 In fact, it is heavily advised to have query string caching with Cloudflare
 enabled in all situations.
 */
	MEDIA_URL: '../',
/*
 If serving static assets from a different subdomain, the links in figcaption
 will not download the image with the original file name. This is due to Cross
 Origin Secutity Policy. As a workaround, you can define a secondary static
 resource URL, that will be only used for downloading messages from these links.
 That way you can still have the benifits of serving statics from a separate
 domain and named image downloads. If left as null, defaults to MEDIA_URL.
 */
	SECONDARY_MEDIA_URL: null,
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

// Default to primary URL
if (!config.SECONDARY_MEDIA_URL)
	config.SECONDARY_MEDIA_URL = config.MEDIA_URL;
