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
	locked: 'locked',
	thread_locked: 'This thread is locked.',
	langApplied: 'Language settings applied. The page will now reload.',
	googleSong: 'Click to google song',
	quoted: 'You have been quoted',
	syncwatchStarting: 'Syncwatch starting in 10 seconds',
	finished: 'Finished',
	expander: ['Expand Images', 'Contract Images'],
	uploading: 'Uploading...',
	subject: 'Subject',
	cancel: 'Cancel',
	unknownUpload: 'Unknown upload error',
	unknownResult: 'Unknown result',
	rescan: 'Rescan',

	reports: {
		post: 'Reporting post',
		reporting: 'Reporting...',
		submitted: 'Report submitted!',
		setup: 'Obtaining reCAPTCHA...',
		loadError: 'Couldn\'t load reCATPCHA'
	},

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

	// Websocket syncronisation status
	sync: {
		notSynced: 'Not synched',
		connecting: 'Connecting',
		syncing: 'Syncing',
		synced: 'Synced',
		dropped: 'Dropped',
		reconnecting: 'Reconnecting'
	},

	// Moderation language map
	mod: {
		title: ['Title', 'Display staff title on new posts'],
		clearSelection: ['Clear', 'Clear selected posts'],
		spoilerImages: ['Spoiler', 'Spoiler selected post images'],
		deleteImages: ['Del Img', 'Delete selected post images'],
		deletePosts: ['Del Post', 'Delete selected posts'],
		lockThreads: ['Lock', 'Lock/unlock selected threads'],
		toggleMnemonics: ['Mnemonics', 'Toggle mnemonic display'],
		sendNotification: [
			'Notification',
			'Send notifaction message to all clients'
		],
		ban: ['Ban', 'Ban poster(s) for the selected post(s)'],
		renderPanel: ['Panel', 'Toggle administrator panel display'],
		modLog: ['Log', 'Show moderation log'],
		djPanel: ['DJ', 'DJ tool panel'],
		displayBan: [
			'Display',
			'Append a public \'USER WAS BANNED FOR THIS POST\' message'
		],
		unban: 'Unban',
		banMessage: 'USER WAS BANNED FOR THIS POST',
		placeholders: {
			msg: 'Message',
			days: 'd',
			hours: 'h',
			minutes: 'min',
			reason: 'Reason'
		},
		needReason: 'Must specify reason',

		// Correspond to websocket calls in common/index.js
		7: 'Image spoilered',
		8: 'Image deleted',
		9: 'Post deleted',
		10: 'Thread locked',
		11: 'Thread unlocked',
		12: 'User banned',
		53: 'User unbanned',

		// Formatting function for moderation messages
		formatLog: function (act) {
			var msg = lang.mod[act.kind] + ' by ' + act.ident;
			if (act.reason)
				msg += ' for ' + act.reason;
			return msg;
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
	// 56 minutes ago / in 56 minutes
	ago: function(time, unit, isFuture) {
		var res = lang.pluralize(time, unit);
		if (isFuture)
			res = 'in ' + res;
		else
			res += ' ago';
		return res;
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
