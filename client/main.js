/*
 * Loads the depandancies in order and aggregates exports from various modules
 */

// NOTE: The entire bundle uses strict mode through the Babel transpiler

/*
 * Because we are going to attach listeners to these all over the place, have
 * to load soome core modules in specific order. Also avoids nasty circular
 * dependaancy, by placing some of the exports here and not in child modules.
 */
let $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	radio = require('backbone.radio');

// Register jquery plugins
require('jquery.cookie');
// Bind jQuery to backbone
Backbone.$ = $;

// Central aplication object and message bus
let main = module.exports = radio.channel('main');
_.extend(main, {
	// Bind dependancies to main object for pretier destructuring requires
	$, _, Backbone,
	stackBlur: require('stack-blur'),

	/*
	 Ofload expensive and not that neccessary initialisation logic till
	 after the core modules are started
	 */
	_deferred: [],
	defer(func) {
		main._deferred.push(func);
		return main;
	},
	execDeffered() {
		_.defer(() => {
			let def = this._deferred;
			for (let i = 0, l = def.length; i < l; i++)
				_.defer(def[i]);
		});
	},

	/*
	 These configs really should not be randomly toggled frequently. No need
	 to put them in state.js, as they should not be hot-loaded. Anything
	 that needs to be, can be moved to hot.js. Should prevent some bugs, but
	 also reduce flexibility, for frequent togglers. Hmm.
	 */
	config: window.config,
	isMobile: /Android|iP(?:hone|ad|od)|Windows Phone/.test(navigator.userAgent),
	// Websocket call handler map. Store them here, to avoid requiring
	// modules in the wrong order.
	dispatcher: {},
	// Read-only boards get expanded later
	readOnly: [],
	memory: require('./memory'),
	lang: require('lang')
});

// You can invoke the client-side debug mode with the `debug=true` query string
if (/[&\?]debug=true/.test(location.href))
	main.config.DEBUG = true;
if (main.config.DEBUG) {
	// Export Backbone instance for easier debugging
	window.Backbone = Backbone;
	// Log all channel traffic
	radio.DEBUG = true;
	radio.tuneIn('main');
}

/*
 Core modules. The other will be more or less decoupled, but these are the
 monolithic foundation.
 */
let state = main.state = require('./state');
let	common = main.common = require('../common');
// Initialise main rendering object
let oneeSama = main.oneeSama = new common.OneeSama({
	op: state.page.get('thread'),
	lang: main.lang,
	// Core post link handler
	tamashii(num) {
		let frag;
		if (this.links && num in this.links) {
			let model = state.posts.get(num);
			const desc = model && model.get('mine') && '(You)';
			frag = this.postRef(num, this.links[num], desc);
		}
		else
			frag = '>>' + num;
		this.callback(frag);
	}
});
main.options = require('./options');
state.page.set('tabID', common.random_id());

_.extend(main, {
	// Cached jQuery objects
	$doc: $(document),
	$threads: $('threads'),
	$name: $('input[name=name]'),
	$email: $('input[name=email]'),

	connSM: new common.FSM('load'),
	postSM: new common.FSM('none')
});

// 2nd tier dependacy modules. These are needed before the websocket
// connection is opened, because they populate the dispatcher handler object.
main.etc = require('./etc');
_.extend(main, {
	loop: require('./loop'),
	time: require('./time'),
	scroll: require('./scroll'),
	notify: require('./notify'),
	banner: require('./banner'),
	report: require('./report'),
	amusement: require('./amusement')
});

// Load post models and views
main.posts = require('./posts');
main.Extract = require('./extract');
// Start the client
main.client = require('./client');
main.conection = require('./connection');

// Load independant auxilary modules
_.extend(main, {
	background: require('./background'),
	history: require('./history'),
	hover: require('./hover'),
	drop: require('./drop'),
	mobile: require('./mobile'),
	hide: require('./hide')
});

main.execDeffered();
