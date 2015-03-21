/*
 * Houses both the actual options controler and the options panel renderring
 * logic
 */

var _ = require('underscore'),
	$ = require('jquery'),
	Backbone = require('backbone'),
	background = require('./background'),
	// Will replace with './client' once stable
	common = require('../common'),
	main = require('./main'),
	state = require('./state');

// Try to get options from local storage
var options;
try {
	options = JSON.parse(localStorage.options);
}
catch(e) {
}
if (!options)
	options = {};
options = exports = new Backbone.Model(options);

// Persists entire model to localStorage on change
options.on('change', function() {
	try {
		localStorage.options = JSON.stringify(options);
	}
	catch(e) {
	}
});

var optionsCollection = new Backbone.Collection(),
	tabs = ['General', 'Style', 'ImageSearch', 'Fun', 'Shortcuts'];

// Controller template for each individual option
var OptionModel = Backbone.Model.extend({
	initialize: function(obj) {
		// Condition for loading option. Optional.
		if (obj.load !== undefined && !obj.load)
			return;
		this.set(obj);
		// No type = checkbox + default false
		if (!this.type) {
			this.type = 'checkbox';
			this.default = false;
		}
		optionsCollection.add(this);
		// Different value for each board
		if (this.exec !== undefined) {
			var id = this.id;
			if (this.boardSpecific)
				id = boardify(id);
			this.listenTo(options, {}['change:' + id] = this.exec);
		}
	}
});

function boardify(id) {
	return 'board.' + state.page.get('board') + '.' + id;
}

/* LAST N CONFIG */
new OptionModel({
	// Key name in the options model
	id: 'lastn',
	// Displayed label in the options panel
	label: '[Last #]',
	// Type of toggle
	type: 'number',
	// Hover tooltip
	tooltip: 'Number of posts to display with the "Last n" thread expansion link',
	// Tab of options panel
	tab: 'General',
	// Function for assesing if value is valid. Optional.
	validation: common.reasonable_last_n,
	'default': state.hotCsonfig.get('THREAD_LAST_N'),
	// Function to execute on change. Optional.
	exec: function(n) {
		main.oneeSama.lastN = n;
		$.cookie('lastn', n, {path: '/'});
	}
});
/* KEEP THREAD LENGTH WITHIN LASTN */
new OptionModel({
	id: 'postUnloading',
	label: 'Dynamic Post Unloading',
	tooltip: 'Improves thread responsiveness by unloading posts from the top of'
		+ ' the thread, so that post count stays within the Last # value. Only'
		+ ' applies to Last # enabled threads',
	tab: 'General',
});
/* LOCK TO BOTTOM EVEN WHEN DOCUMENT HIDDEN*/
new OptionModel({
	id: 'alwaysLock',
	label: 'Always Lock to Bottom',
	tootltip: 'Lock scrolling to page bottom even when tab is hidden',
	tab: 'General',
});
/* THEMES */
new OptionModel({
	id: 'theme',
	boardSpecific: true,
	label: 'Theme',
	// Arrays will turn into selection boxes
	type: [
		'moe',
		'gar',
		'mawaru',
		'moon',
		'ashita',
		'console',
		'tea',
		'higan',
		'rave',
		'tavern',
		'glass'
	],
	tooltip: 'Select CSS theme',
	tab: 'Style',
	'default': state.hotConfig.get('BOARD_CSS')[state.page.get('board')],
	exec: function(theme) {
		if (theme) {
			var css = hotConfig.css[theme + '.css'];
			$('#theme').attr('href', state.imagerConfig.get('mediaURL')
				+ 'css/' + css);
		}
		// Call the background controller to generate, remove and/or append the glass
		background.glass(theme);
	}
});
/* THUMBNAIL OPTIONS */
new OptionModel({
	id: 'thumbs',
	label: 'Thumbnails',
	type: common.thumbStyles,
	tooltip: 'Set thumbnail type: '
		+ 'Small: 125x125, small file size; '
		+ 'Sharp: 125x125, more detailed; '
		+ 'Hide: hide all images;',
	tab: 'style',
	'default': 'small',
	exec: function(type) {
		$.cookie('thumb', type);
		main.oneeSama.thumbStyle = type;
	}
});
/* REPLY AT RIGHT */
new OptionModel({
	id: 'replyright',
	label: '[Reply] at Right',
	tooltip: 'Move Reply button to the right side of the page',
	tab: 'Style',
	exec: function(r) {
		if (r)
			$('<style/>', {
				id: 'reply-at-right',
				text: 'aside { margin: -26px 0 2px auto; }',
			}).appendTo('head');
		else
			$('#reply-at-right').remove();
	}
});
/* BACKLINKS */
new OptionModel({
	id: 'nobacklinks',
	label: 'Backlinks',
	type: 'checkbox',
	tooltip: 'Links to replies of current post',
	tab: 'General',
	'default': true,
	// TODO: Implement backlinks in models.js
	exec: function() {}
});
/* LINKIFY TEXT URLS */
new OptionModel({
	id: 'linkify',
	label: 'Linkify text URLs',
	tooltip: 'Convert in-post text URLs to clickable links. WARNING: Potential'
		+ ' security hazard (XSS). Requires page refresh.',
	tab: 'General',
	exec: function(toggle) {
		$.cookie('linkify', toggle, {path: '/'});
	}
});
/* ANONIMISE ALL POSTER NAMES */
new OptionModel({
	id: 'anonymise',
	label: 'Anonymise',
	tooltip: 'Display all posters as anonymous',
	tab: 'General'
});
/* RELATIVE POST TIMESTAMPS */
new OptionModel({
	id: 'relativeTime',
	label: 'Relative Timestamps',
	tooltip: 'Relative post timestamps. Ex.: "1 hour ago." Requires page'
		+ ' refresh',
	tab: 'General',
	exec: function(toggle) {
		$.cookie('rTime', toggle, {path: '/'});
	}
});
/* R/A/DIO NOW PLAYING BANNER */
new OptionModel({
	id: 'nowPlaying',
	label: 'Now Playing Banner',
	type: 'checkbox',
	tooltip: 'Currently playing song on r/a/dio and other stream information in'
		+ ' the top banner.',
	tab: 'Fun',
	'default': true,
	exec: function(toggle) {
		if (toggle)
			banner.view.clearRadio();
		// Query the server for current stream info
		else
			main.send([common.RADIO]);
	}
});
/* IMAGE SEARCH LINK TOGGLE */
var search = ['google', 'iqdb', 'saucenao', 'foolz', 'exhentai'],
	image, capital;
for (var i = 0; i < search.length; i++) {
	image = search[i];
	capital = image[0].toUpperCase() + image.slice(1);
	$('<style/>', {id: image + 'Toggle'})
		.html('.' + image + '{display:none;}')
		.appendTo('head');

	new OptionModel({
		id: image,
		label: capital + ' Image Search',
		tooltip: 'Show/Hide ' + capital + ' image search links',
		tab: 'ImageSearch',
		exec: function(toggle) {
			$('#' + image + 'Toggle').prop('disabled', toggle);
		}
	});
}
/* SPOILER TOGGLE */
new OptionModel({
	id: 'noSpoilers',
	label: 'Image Spoilers',
	type: 'checkbox',
	tooltip: "Don't spoiler images",
	tab: 'Style',
	'default': true,
	exec: function(spoilertoggle) {
		$.cookie('spoil', spoilertoggle, {path: '/'});
		oneeSama.spoilToggle = spoilertoggle;
	}
});
/* Autogif TOGGLE */
new OptionModel({
	id: 'autogif',
	label: 'Animated GIF Thumbnails',
	tooltip: 'Animate GIF thumbnails',
	tab: 'Style',
	exec: function(autogif) {
		$.cookie('agif', autogif, {path: '/'});
		oneeSama.autoGif = autogif;
	}
});
/* NOTIFICATIONS */
new OptionModel({
	id: 'notification',
	label: 'Desktop Notifications',
	tooltip: 'Get desktop notifications when quoted or a syncwatch is about to'
		+ ' start',
	tab: 'General',
	exec: function(notifToggle) {
		if (notifToggle && (Notification.permission !== "granted"))
			Notification.requestPermission();
	}
});
/* ILLYA DANCE */
var illyaDance = new OptionModel({
	id: 'illyaBGToggle',
	label: 'Illya Dance',
	tooltip: 'Dancing loli in the background',
	tab: 'Fun',
	exec: function(illyatoggle) {
		var muted = ' ';
		if (options.get(option_illya_mute.id))
			muted = 'muted';
		var dancer = '<video autoplay ' + muted + ' loop id="bgvid" >' +
			'<source src="' + mediaURL + 'illya.webm" type="video/webm">' +
			'<source src="' + mediaURL + 'illya.mp4" type="video/mp4">' +
			'</video>';
		if (illyatoggle)
			$("body").append(dancer);
		else
			$("#bgvid").remove();
	}
});
new OptionModel({
	id: 'illyaMuteToggle',
	label: 'Mute Illya',
	tooltip: 'Mute dancing loli',
	tab: 'Fun',
	exec: function option_illya_mute(toggle) {
		if (options.get(illyaBGToggle)) {
			illyaDance.exec(false);
			illyaDance.exec(true);
		}
	}
});
/* HORIZONTAL POSTING */
new OptionModel({
	id: 'horizontalPosting',
	label: 'Horizontal Posting',
	tooltip: '38chan nostalgia',
	tab: 'Fun',
	exec: function(toggle) {
		var style = '<style id="horizontal">article,aside{display:inline-block;}</style>';
		if (toggle)
			$('head').append(style);
		else
			$('#horizontal').remove();
	}
});
/* CUSTOM USER-SET BACKGROUND */
new OptionModel({
	id: 'userBG',
	label: 'Custom Background',
	tooltip: 'Toggle custom page background',
	tab: 'Style',
});
new OptionModel({
	id: 'userBGimage',
	label: '',
	type: 'image',
	tooltip: "Image to use as the background",
	tab: 'Style',
	exec: background.set
});
/* IMAGE HOVER EXPANSION */
new OptionModel({
	id: 'imageHover',
	label: 'Image Hover Expansion',
	tooltip: 'Display image previews on hover',
	tab: 'General',
});
new OptionModel({
	id: 'webmHover',
	label: 'WebM Hover Expansion',
	tooltip: 'Display WebM previews on hover. Requires Image Hover Expansion'
		+ ' enabled.',
	tab: 'General'
});
/* INLINE EXPANSION */
new OptionModel({
	id: 'inlinefit',
	label: 'Expansion',
	type: ['none', 'full-size', 'fit to width', 'fit to height',
		'fit to both'],
	tooltip: 'Expand images inside the parent post and resize according to'
		+ ' setting',
	tab: 'Style',
	'default': 'width'
});
/* SHORTCUT KEYS */
// NOTE: also use loop
/*new OptionModel({
 id:
 label:
 type:
 tooltip:
 tab:
 'default':
 exec:
 });
 new OptionModel({
 id:
 label:
 type:
 tooltip:
 tab:
 'default':
 exec:
 });
 new OptionModel({
 id:
 label:
 type:
 tooltip:
 tab:
 'default':
 exec:
 });
 new OptionModel({
 id:
 label:
 type:
 tooltip:
 tab:
 'default':
 exec:
 });
 new OptionModel({
 id:
 label:
 type:
 tooltip:
 tab:
 'default':
 exec:
 });
 */