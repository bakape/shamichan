/*
This file can be hot-loaded into a running server with `node server/kill.js`.
 */

this.hot = {
// User feedback email
	EMAIL: "lalc@doushio.com",
	TITLES: {
		moe: "/moe/ - Sweets",
		gar: "/gar/ - Hard Work &amp; Guts",
		meta: "/meta/ - The Abyss",
		staff: "/staff/"
	},
// Default theme to use
	DEFAULT_CSS: 'moe',
/*
 * File names of the images to use as banners inside the ./www/banners
 * Example: ['banner01.png', 'banner02.gif', 'banner03.jpg'] or null
 */
	BANNERS: null,

// Instead of redirecting to the default board serve a frontpage to the
// user, when navigating to '/'.Must be path pointing to a regular HTML
// document or null. Example: 'www/frontpage.html'
	frontpage: null,

	THREADS_PER_PAGE: 10,
// Replies to display under each thread on the board's root page
	ABBREVIATED_REPLIES: 5,
/*
 Default number of posts to display, when thread is expanded with the "Last N"
 link
 */
	THREAD_LAST_N: 100,
	SUBJECT_MAX_LENGTH: 50,
	EXCLUDE_REGEXP: /[\u2000-\u200f\u202a-\u202f\u205f-\u206f]+/g,
	SAGE_ENABLED: true,
// Disable names and  for new posts
	forced_anon: false,
// Boards that won't be displayed in the banner board navigation
	hidden_boards: [],
// Titles for staff that will be displayed in their posts' headers
	staff_aliases: {
		admin: 'Admin',
		moderator: 'Moderator',
		dj: 'DJ',
		janitor: 'Janitor'
	},
	SPECIAL_TRIPCODES: {
		kyubey: "／人◕ ‿‿ ◕人＼"
	},
/*
 Information to display in the top banner. Accepts HTML. Is overriden by
 Y.set_banner()
 */
	BANNERINFO: '',
// Planned event schedule to display in the banner's Schedule list
	SCHEDULE: [
		'Mon', null, null,
		'Tue', null, null,
		'Wed', null, null,
		'Thu', null, null,
		'Fri', null, null,
		'Sat', null, null,
		'Sun', null, null
	],
// Entries for the banner's FAQ list
	FAQ: [
		'Upload size limit is 100 MB',
		'Accepted upload file types: JPG, JPEG, PNG, APNG, GIF, WEBM, SVG,'
			+ ' PDF, MP3(must have cover art)',
		'<hr>',
		'Hash commands: ',
		'#&#60;number of dice(1-10, optional)&#62;d&#60;dice sides(1-100)&#62;'
			+ ' - Roll dice',
		'#flip - Coinflip',
		'#8ball - An 8ball',
		'#pyu - Missle launcher',
		'#pcount - Launch count',
		'#q - Print r/a/dio song queue',
		'#sw&#60;hours(optional)&#62;:&#60;minutes&#62;:&#60;seconds&#62;[+-]'
			+ '&#60;offset seconds(optional)&#62 - Syncronised duration timer',
		'&emsp;A positive offset adds a countdown. A negative offset starts'
			+ ' the timer n seconds into the episode.',
		'<hr>',
		'Source code repository: <a href="https://github.com/bakape/doushio"'
			+ ' target="_blank">github.com/bakape/meguca</a>'
	],
// Extra JS script to load on all clients. Set to a file path or null.
	inject_js: null,
/*
 Word replacament filter. {p: /foo/, r: 'bar'} Pattern must not contain spaces
 or newlines
 */
	FILTER: [],
/*
 Array of answers for the 8ball random wisdom dispenser. To use, type "#8ball"
 in post, followed by enter.
 */
	EIGHT_BALL: [
		"Yes",
		"No",
		"Maybe",
		"It can't be helped",
		"Hell yeah, motherfucker",
		'Ara ara~',
		"That is my fetish",
		"Anta baka?"
	],
/*
 Local http://loli.dance/ implementation. Videos not included in git tree.
 Place illya.webm and illya.mp4 into the www directory, if you want this.
 */
	ILLYA_DANCE: false
};
