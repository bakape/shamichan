/*
 * Common methods to both OP and regular posts
 */

const main = require('../main'),
	imager = require('./imager'),
	{$, _, common, etc, lang, oneeSama, options, state} = main;

module.exports = {
	initialize() {
		this.listenTo(this.model, 'dispatch', this.redirect);
	},
	// Extra initialisation logic for posts renderred client-side
	clientInit() {
		if (options.get('anonymise'))
			this.anonymise();
		return this;
	},
	// Proxy to the appropriate method
	redirect(command, ...args) {
		this[command](...args);
	},
	updateBody(update) {
		if (!this.$blockquote)
			this.$blockquote = this.$el.find('blockquote').first();
		oneeSama.dice = update.dice;
		oneeSama.state = update.state;
		oneeSama.callback = this.inject;
		oneeSama.$buffer = this.$blockquote;
		oneeSama.fragment(update.frag);
	},
	// Inject various tags into the blockqoute
	inject(frag) {
		var $dest = this.$buffer;
		for (var i = 0; i < this.state[1]; i++)
			$dest = $dest.children('del').last();
		if (this.state[0] == common.S_QUOTE)
			$dest = $dest.children('em').last();
		if (this.strong)
			$dest = $dest.children('strong').last();
		var out = null;
		if (frag.safe) {
			var m = frag.safe.match(/^<(\w+)>$/);
			if (m)
				out = document.createElement(m[1]);
			else if (/^<\/\w+>$/.test(frag.safe))
				out = '';
		}
		if (out === null) {
			if (Array.isArray(frag))
				out = $(common.flatten(frag).join(''));
			else
				out = common.escape_fragment(frag);
		}
		if (out)
			$dest.append(out);
		return out;
	},
	renderTime() {
		this.el.query('time').outerHTML = oneeSama.time(this.model.get('time'));
	},
	renderBacklinks(links) {
		main.follow(() =>
			this.el.query('small').innerHTML = oneeSama.backlinks(links));
	},
	// Admin JS injections
	fun() {
		// Fun goes here
	},
	// Self-delusion tripfag filter
	anonymise() {
		this.el.query('.name').innerHTML = `<b class="name">${lang.anon}<b>`;
	},
	// Restore regular name
	renderName() {
		this.el.query('name').outerHTML = oneeSama.name(this.model.attributes);
	},
	renderModerationInfo(info) {
		const el = this.getContainer();
		el.query('.modlog').remove();
		el.query('blockquote').before(etc.paseDOM(oneeSama.modInfo(info)));
	},
	getContainer() {
		return this.el.query('.container');
	},
	renderBan() {
		const el = this.getContainer();
		el.query('.banMessage').remove();
		el.query('blockquote').after(etc.parseDOM(oneeSama.banned()));
	},
	renderEditing(editing) {
		const {el} = this;
		if (editing)
			el.classList.add('editing');
		else {
			el.classList.remove('editing');
			el.query('blockquote').normalize();
		}
	}
};

_.extend(module.exports, imager.Hidamari);
