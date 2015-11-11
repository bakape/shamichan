/*
 * Server-only mapping of configurable language options
 */
var lang = {
	catalog_omit: 'Replies/Images',
	show_seconds: 'Click to show seconds',
	worksBestWith: 'works best with',

	// Imager responses
	im : {
		bad_client: "Bad client ID.",
		too_large: 'File is too large.',
		req_problem: 'Upload request problem.',
		aborted: 'Upload was aborted.',
		received: '% received...',
		invalid: 'Invalid upload.',
		no_image: 'No image.',
		bad_spoiler: 'Bad spoiler.',
		temp_tracking: "Temp tracking error: ",
		invalid_format: 'Invalid image format.',
		verifying: 'Verifying...',
		missing: "File went missing.",
		video_invalid: "Invalid video file.",
		ffmpeg_too_old: "Server's ffmpeg is too old.",
		mp3_no_cover: 'MP3 has no cover art.',
		video_unknown: "Unknown video reading error.",
		video_format: 'File format corrupted.',
		audio_kinshi: 'Audio is not allowed.',
		bad: 'Bad image.',
		not_png: 'Not PNG or APNG.',
		video: 'Video',
		image: 'Image',
		bad_dims: 'Bad image dimensions.',
		too_many_pixels: 'Way too many pixels.',
		too_wide_and_tall: ' is too wide and too tall.',
		too_wide: ' is too wide.', // No such thing
		too_tall: ' is too tall.',
		thumbnailing: 'Thumbnailing...',
		not_image: 'File is not an image',
		unsupported: "Unsupported file type.",
		dims_fail: "Couldn't read image dimensions.",
		hashing: 'Hashing error.',
		resizing: "Resizing error.",
		pngquant: "Pngquant thumbnailing error.",
		unknown: 'Unknown image processing error.'
	},

	//Various template strings
	tmpl: {
		name: 'Name:',
		email: 'Email:',
		options: 'Options',
		identity: 'Identity',
		faq: 'FAQ',
		schedule: 'Schedule',
		feedback: 'Feedback',
		onlineCounter: 'Online Counter'
	},

	/*
	 * Client options. The options panel is rendered on template generation, so
	 * these are only needed by the server.
	 * id: [label, tooltip]
	 */
	opts: {
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

		// Names for the options panel tabs
		tabs: ['General', 'Style', 'ImageSearch', 'Fun', 'Shortcuts'],
		export: [
			'Export',
			'Export options to file'
		],
		import: [
			'Import',
			'Import options from file'
		],
		hidden: [
			'Hidden: 0',
			'Clear hidden posts'
		],
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
				+ 'Hide: hide all images;'
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
		spoilers: [
			'Image Spoilers',
			"Don't spoiler images"
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
			'Relative post timestamps. Ex.: \'1 hour ago\''
		],
		nowPlaying: [
			'Now Playing Banner',
			'Currently playing song on r/a/dio and other stream information in'
				+ ' the top banner.'
		],
		// Custom localisation functions
		imageSearch: [
			function(site) {
				return lang.common.capitalize(site)  + ' Image Search';
			},
			function(site) {
				return `Show/Hide ${lang.common.capitalize(site)} search links`;
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
				+ ' Only applies to Last # enabled threads'
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
		],
		workMode: [
			'Work mode',
			'Hides images, defaults theme and disables user background'
		],
		workModeTOG: [
			'Work mode',
			'Hides images, defaults theme and disables user background'
		]
	}
};

lang.common = require('./common');

module.exports = lang;
