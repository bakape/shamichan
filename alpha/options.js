/*
 * Houses both the actual options controler and the options panel renderring
 * logic
 */

var _ = require('underscore'),
	$ = require('jquery'),
	Backbone = require('backbone'),
	background = require('./background'),
	banner = require('./banner'),
	common = require('../common'),
	main = require('./main'),
	state = require('./state');

var notMobile = !main.isMobile;

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
		if (!obj.type)
			this.set({type: 'checkbox', 'default': false});
		optionsCollection.add(this);
		if (obj.exec !== undefined) {
			var id = obj.id;
			// Different value for each board
			if (obj.boardSpecific) {
				id = boardify(id);
				this.set('id', id);
			}
			var opts = {};
			opts['change:' + id] = obj.exec;
			this.listenTo(options, opts);
		}
	},
	// The untampered id, we get before boardification
	initialId: function() {
		var id = this.get('id');
		return this.get('boardSpecific') ? id.slice('.')[2] : id;
	}
});

function boardify(id) {
	return 'board.' + state.page.get('board') + '.' + id;
}

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
/* THUMBNAIL OPTIONS */
new OptionModel({
	id: 'thumbs',
	boardSpecific: true,
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
/* IMAGE HOVER EXPANSION */
new OptionModel({
	id: 'imageHover',
	label: 'Image Hover Expansion',
	load: notMobile,
	tooltip: 'Display image previews on hover',
	tab: 'General',
});
new OptionModel({
	id: 'webmHover',
	label: 'WebM Hover Expansion',
	load: notMobile,
	tooltip: 'Display WebM previews on hover. Requires Image Hover Expansion'
		+ ' enabled.',
	tab: 'General'
});
/* Autogif TOGGLE */
new OptionModel({
	id: 'autogif',
	load: notMobile,
	label: 'Animated GIF Thumbnails',
	tooltip: 'Animate GIF thumbnails',
	tab: 'Style',
	exec: function(autogif) {
		$.cookie('agif', autogif, {path: '/'});
		oneeSama.autoGif = autogif;
	}
});
/* SPOILER TOGGLE */
new OptionModel({
	id: 'noSpoilers',
	boardSpecific: true,
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
/* DESKTOP NOTIFICATIONS */
new OptionModel({
	id: 'notification',
	load: notMobile,
	label: 'Desktop Notifications',
	tooltip: 'Get desktop notifications when quoted or a syncwatch is about to'
		+ ' start',
	tab: 'General',
	exec: function(notifToggle) {
		if (notifToggle && (Notification.permission !== "granted"))
			Notification.requestPermission();
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
	load: notMobile,
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
['google', 'iqdb', 'saucenao', 'foolz', 'exhentai'].forEach(function(search) {
	var capital = search[0].toUpperCase() + search.slice(1);
	$('<style/>', {id: search + 'Toggle'})
		.html('.' + search + '{display:none;}')
		.appendTo('head');

	new OptionModel({
		id: search,
		label: capital + ' Image Search',
		tooltip: 'Show/Hide ' + capital + ' search search links',
		tab: 'ImageSearch',
		exec: function(toggle) {
			$('#' + search + 'Toggle').prop('disabled', toggle);
		}
	});
});
/* ILLYA DANCE */
var illyaDance = new OptionModel({
	id: 'illyaBGToggle',
	load: notMobile,
	label: 'Illya Dance',
	boardSpecific: true,
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
	load: notMobile,
	boardSpecific: true,
	label: 'Mute Illya',
	tooltip: 'Mute dancing loli',
	tab: 'Fun',
	exec: function option_illya_mute() {
		if (options.get(illyaBGToggle)) {
			illyaDance.exec(false);
			illyaDance.exec(true);
		}
	}
});
/* HORIZONTAL POSTING */
new OptionModel({
	id: 'horizontalPosting',
	boardSpecific: true,
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
/* CUSTOM USER-SET BACKGROUND */
new OptionModel({
	id: 'userBG',
	load: notMobile,
	label: 'Custom Background',
	tooltip: 'Toggle custom page background',
	tab: 'Style',
});
new OptionModel({
	id: 'userBGimage',
	load: notMobile,
	label: '',
	type: 'image',
	tooltip: "Image to use as the background",
	tab: 'Style',
	exec: background.set
});
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
	'default': state.hotConfig.get('THREAD_LAST_N'),
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
/* SHORTCUT KEYS */
[
	{
		label: 'New Post',
		id: 'new',
		default: 78,
		tooltip: 'Open new post'
	}, {
		label: 'Image Spoiler',
		id: 'togglespoiler',
		default: 73,
		tooltip: 'Toggle spoiler in the open post'
	}, {
		label: 'Text Spoiler',
		id: 'textSpoiler',
		default: 68,
		tooltip: 'Insert text spoiler tag'
	}, {
		label: 'Finish Post',
		id: 'done',
		default: 83,
		tooltip: 'Close open post'
	}, {
		label: 'Expand All Images',
		id: 'expandAll',
		default: 69,
		tooltip: 'Expand all images. Webm, PDF and MP3 and your own post'
			+ ' aren\'t affected. New post images are also expanded.'
	}
].forEach(function(short) {
	short.type = 'shortcut';
	short.tab = 'Shortcuts';
	new OptionModel(short);
});

// Highlight options button, if no options are set
(function() {
	if (localStorage.getItem('options'))
		return;
	var $el = $('#options');
	$el.addClass('noOptions');

	function fadeout() {
		$el.filter('.noOptions').fadeOut(fadein);
	}

	function fadein() {
		$el.fadeIn();
		// Stop animation, if options pannel is opened
		if ($el.filter('.noOptions').length)
			fadeout();
	}

	fadeout();

	$el.click(function() {
		$el.removeClass('noOptions');
	});
})();

// Render options panel
var OptionsView = Backbone.View.extend({
	initialize: function() {
		this.render();
	},
	tagName: 'div',
	className: 'bmodal',
	id: 'options-panel',
	render: function() {
		var $tabSel = $('<ul/>', {'class': 'option_tab_sel'}),
			$tabCont = $('<ul/>', {'class': 'option_tab_cont'});
		// Render tabs
		tabs.forEach(function(tab) {
			$('<li/>').append(
				$('<a/>', {
					'data-content': tab,
					href: '#' + tab,
					'class': tab
				})
				.html(tab)
				)
				.appendTo($tabSel);
			$('<li/>', {
				'data-content': tab,
				'class': tab
			})
				.appendTo($tabCont);
		});
		// Render all the options
		var opts = options.attributes;
		optionsCollection.models.forEach(function(model) {
			var model = model.attributes,
				val = opts[model.id] || model.default,
				$tab = $tabCont.children('.' + model.tab),
				$input;

			if (model.type == 'checkbox') {
				$input = $('<input/>', {
					type: 'checkbox',
					checked: val
				});
			}
			else if (model.type == 'number') {
				$input = $('<input/>', {
					width: '4em',
					maxlength: 4,
					val: val
				});
			}
			else if (model.type == 'image') {
				$input = $('<input/>', {
					type: 'file'
				});
			}
			else if (model.type instanceof Array) {
				$input = $('<select/>');
				model.type.forEach(function(opt) {
					$input.append('<option value="' + opt + '">' + opt
						+ '</option>');
				});
			}
			else if (model.type == 'shortcut') {
				$tab.append('Alt+');
				$input = $('<input/>', {
					id: model.id,
					maxlength: 1,
					val: String.fromCharCode(val)
				});
			}
			$tab.append([
				$input.attr({
					id: model.id,
					title: model.tootlip
				}),
				$('<label/>', {
					for : model.id,
					title: model.tooltip
				})
					.html(model.label),
				'<br>'
			]);
		});

		// Show the first tab by default
		$tabSel.children().first().addClass('tab_sel');
		$tabCont.children().first().addClass('tab_sel');

		this.$el.append($tabSel, $tabCont);
		this.$el.appendTo('body');
	},
	events: {
		'click .option_tab_sel>li>a': 'switchTab'
	},
	switchTab: function(event) {
		event.preventDefault();
		var $a = $(event.target);
		// Unhighight all tabs1
		this.$el.children('.option_tab_sel').find('a').removeClass('tab_sel');
		// Hightlight the new one
		$a.addClass('tab_sel');
		// Switch tabs
		var $li = this.$el.children('.option_tab_cont').children('li');
		$li.removeClass('tab_sel');
		$li.filter('.' + $a.data('content')).addClass('tab_sel');
	}
});

var optionsView;
// Rander it after the current stack clears,for a bit more responsiveness
_.defer(function() {
	optionsView = new OptionsView();
});

// TODO: BoardSpecific option unloading code for inter-board push state navigation