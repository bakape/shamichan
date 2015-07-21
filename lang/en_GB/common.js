/*
 * Shared by the server and client
 */

var lang = {
	anon: 'Anonymous',
	search: 'Search',
	show: 'Show',
	hide: 'Hide',
	report: 'Report',
	focus: 'Focus',
	expand: 'Expand',
	last: 'Last',
	see_all: 'See all',
	bottom: 'Bottom',
	expand_images: 'Expand Images',
	live: 'live',
	catalog: 'Catalog',
	return: 'Return',
	top: 'Top',
	reply: 'Reply',
	newThread: 'New thread',
	locked_to_bottom: 'Locked to bottom',
	you: '(You)',
	done: 'Done',
	send: 'Send',

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

	// Moderation language map
	mod: {
		clearSelection: ['Clear', 'Clear selected posts'],
		spoilerImages: ['Spoiler', 'Spoiler selected post images'],
		deleteImages: ['Del Img', 'Delete selected post images'],
		deletePosts: ['Del Post', 'Delete selected posts'],
		lockThread: ['Lock', 'Lock selected threads'],
		toggleMnemonics: ['Mnemonics', 'Toggle mnemonic display'],
		sendNotification: [
			'Notification',
			'Send notifaction message to all clients'
		],
		dispatchFun: ['Fun', 'Execute arbitrary JavaScript on all clients'],
		renderPanel: ['Panel', 'Toggle administrator panel display'],
		imgDeleted: 'Image Deleted',
		placeholders: {
			msg: 'Message'
		}
	},

	// Format functions
	pluralize: function(n, noun) {
		// For words ending with 'y' and not a vovel before that
		if (n != 1
			&& noun.slice(-1) == 'y'
			&& ['a', 'e', 'i', 'o', 'u'].indexOf(noun.slice(-2, -1)
				.toLowerCase()) < 0) {
			noun = noun.slice(0, -1) + 'ie';
		}
		return n + ' ' + noun + (n == 1 ? '' : 's');
	},
	capitalize: function(word) {
		return word[0].toUpperCase() + word.slice(1);
	},
	// 56 minutes ago
	ago: function(time, unit) {
		return lang.pluralize(time, unit) + ' ago';
	},
	// 47 replies and 21 images omitted
	abbrev_msg:  function(omit, img_omit, url) {
		var html = lang.pluralize(omit, 'reply');
		if (img_omit)
			html += ' and ' + lang.pluralize(img_omit, 'image');
		html += ' omitted';
		if (url) {
			html += ' <span class="act"><a href="' + url + '" class="history">'
				+ lang.see_all + '</a></span>';
		}
		return html;
	}
};

module.exports = lang;
