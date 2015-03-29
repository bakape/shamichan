/*
 * Mapping of configurable language options
 */
const lang = {
	anon: 'Anonymous',
	search: 'Search',
	show: 'Show',
	expand: 'Expand',
	last: 'Last',
	see_all: 'See all',
	bottom: 'Bottom',
	expand_images: 'Expand Images',
	live: 'live',
	catalog: 'Catalog',
	return: 'Return',
	top: 'Top',

	// Thumbnail styles
	small: 'small',
	sharp: 'sharp',
	hide: 'hide',
	// Image fit modes
	none: 'none',
	full: 'full-size',
	width: 'fit to width',
	height: 'fit to height',
	both: 'fit to both',

	// Time-related
	week: ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'],
	year: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep',
		'Oct', 'Nov', 'Dec'],
	just_now: 'just now',
	unit_minute: 'minute',
	unit_hour: 'hour',
	unit_day: 'day',
	unit_month: 'month',
	unit_year: 'year',

	// Format functions
	pluralize: function(n, noun) {
		// For words ending with 'y'
		if (n != 1 && noun.slice(-1) == 'y')
			noun = noun.slice(0, -1) + 'ie';
		return n + ' ' + noun + (n == 1 ? '' : 's');
	},
	capitalize: function(word) {
		return word[0].toUpperCase() + word.slice(1);
	},
	// 56 minutes ago
	ago: function(time, unit) {
		return lang.pluralize(time, unit) + ' ago';
	},
	// 47 replies and 21 images omited
	abbrev_msg:  function(omit, img_omit) {
		return lang.pluralize(omit, 'reply')
			+ (img_omit ? ' and ' + lang.pluralize(img_omit, 'image') : '')
			+ ' omitted.';
	},

	// Imager responses
	im_bad_client: "Bad client ID.",
	im_too_large: 'File is too large.',
	im_req_problem: 'Upload request problem.',
	im_aborted: 'Upload was aborted.',
	im_received: '% received...',
	im_invalid: 'Invalid upload.',
	im_no_image: 'No image.',
	im_bad_spoiler: 'Bad spoiler.',
	im_temp_tracking: "Temp tracking error: ",
	im_invalid_format: 'Invalid image format.',
	im_verifying: 'Verifying...',
	im_missing: "File went missing.",
	im_video_invalid: "Invalid video file.",
	im_ffmpeg_too_old: "Server's ffmpeg is too old.",
	im_mp3_no_cover: 'MP3 has no cover art.',
	im_video_unknown: "Unknown video reading error.",
	im_video_format: 'File format corrupted.',
	im_audio_kinshi: 'Audio is not allowed.',
	im_bad: 'Bad image.',
	im_not_png: 'Not PNG or APNG.',
	im_video: 'Video',
	im_image: 'Image',
	im_bad_dims: 'Bad image dimensions.',
	im_too_many_pixels: 'Way too many pixels.',
	im_too_wide_and_tall: ' is too wide and too tall.',
	im_too_wide: ' is too wide.', // No such thing
	im_too_tall: ' is too tall.',
	im_thumbnailing: 'Thumbnailing...',
	im_not_image: 'File is not an image',
	im_unsupported: "Unsupported file type.",
	im_dims_fail: "Couldn't read image dimensions.",
	im_hashing: 'Hashing error.',
	im_resizing: "Resizing error.",
	im_pngquant: "Pngquant thumbnailing error.",
	im_unknown: 'Unknown image processing error.',

	// Client-only

	// Options; id: [label, tooltip]
	opts: {
		lang: [
			'Language',
			'Change interface language'
		],
		inlinefit: [
			'Expansion',
			'Expand images inside the parent post and resize according to setting'
		],
		thumbs: [
			'Thumbnails',
			'Set thumbnail type: '
				+ 'Small: 125x125, small file size; '
				+ 'Sharp: 125x125, more detailed; '
				+ 'Hide: hide all images;',
		],
		imageHover: [
			'Image Hover Expansion',
			'Display image previews on hover'
		],
		webmHover: [
			'WebM Hover Expansion',
			'Display WebM previews on hover. Requires Image Hover Expansion enabled.'
		],
		autogif: [
			'Animated GIF Thumbnails',
			'Animate GIF thumbnails'
		],
		noSpoilers: [
			'Image Spoilers',
			"Don't spoiler images"
		],
		nobacklinks: [
			'Backlinks',
			'Links to replies of current post'
		],
		linkify: [
			'Linkify text URLs',
			'Convert in-post text URLs to clickable links. WARNING: Potential'
				+ ' security hazard (XSS). Requires page refresh.'
		],
		notification: [
			'Desktop Notifications',
			'Get desktop notifications when quoted or a syncwatch is about to start'
		],
		anonymise: [
			'Anonymise',
			'Display all posters as anonymous'
		],
		relativeTime: [
			'Relative Timestamps',
			'Relative post timestamps. Ex.: "1 hour ago." Requires page refresh'
		],
		nowPlaying: [
			'Now Playing Banner',
			'Currently playing song on r/a/dio and other stream information in'
				+ ' the top banner.'
		],
		// Custom localisation functions
		imageSearch: [
			function(site) {
				return lang.capitalize(site)  + ' Image Search';
			},
			function(site) {
				return 'Show/Hide ' + lang.capitalize(site) + ' search search links';
			}
		],
		illyaBGToggle: [
			'Illya Dance',
			'Dancing loli in the background'
		],
		illyaMuteToggle: [
			'Mute Illya',
			'Mute dancing loli'
		],
		horizontalPosting: [
			'Horizontal Posting',
			'38chan nostalgia'
		],
		replyright: [
			'[Reply] at Right',
			'Move Reply button to the right side of the page'
		],
		theme: [
			'Theme',
			'Select CSS theme'
		],
		userBG: [
			'Custom Background',
			'Toggle custom page background'
		],
		userBGimage: [
			'',
			"Image to use as the background"
		],
		lastn: [
			'[Last #]',
			'Number of posts to display with the "Last n" thread expansion link'
		],
		postUnloading: [
			'Dynamic Post Unloading',
			'Improves thread responsiveness by unloading posts from the top of'
				+ ' the thread, so that post count stays within the Last # value.'
				+ ' Only applies to Last # enabled threads',
		],
		alwaysLock: [
			'Always Lock to Bottom',
			'Lock scrolling to page bottom even when tab is hidden'
		],
		// Shortcut keys
		new: [
			'New Post',
			'Open new post'
		],
		togglespoiler: [
			'Image Spoiler',
			'Toggle spoiler in the open post'
		],
		textSpoiler: [
			'Text Spoiler',
			'Insert text spoiler tag'
		],
		done: [
			'Finish Post',
			'Close open post'
		],
		expandAll: [
			'Expand All Images',
			'Expand all images. Webm, PDF and MP3 and your own post'
				+ ' aren\'t affected. New post images are also expanded.'
		]
	}

};

module.exports = lang;