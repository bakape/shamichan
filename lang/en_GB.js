/*
 * Mapping of configurable language options
 */
var lang = {
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
	// 56 minutes ago
	ago: function(time, unit) {
		return lang.pluralize(time, unit) + ' ago';
	},
	// 47 replies and 21 images omited
	abbrev_msg:  function(omit, img_omit) {
		return omit + lang.pluralize(omit, 'reply')
			+ (img_omit ? ' and ' + lang.pluralize(img_omit, 'image') : '')
			+ ' omitted.';
	}
};

module.exports = lang;