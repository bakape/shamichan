/*
 * Common methods to both OP and regular posts
 */

let main = require('../main'),
	imager = require('./imager'),
	Menu = require('./menu'),
	{$, _, common, lang, oneeSama, options, state} = main;

module.exports = {
	events: {
		'click >figure>figcaption>.imageToggle': 'toggleThumbnailVisibility',
		'click >figure>a': 'imageClicked',
		'click >header>nav>a.quote': 'quotePost',
		'click >header>.control': 'renderMenu'
	},
	initCommon() {
		this.$blockquote = this.$el.children('blockquote');
		this
			.listenTo(this.model, 'dispatch', this.redirect)
			.listenTo(state.linkerCore,
				'change:' + this.model.get('num'),
				this.renderBacklinks
			);
		this.fun();
		const links = state.linkerCore.get(this.model.get('num'));
		if (links)
			this.renderBacklinks(null, links);
		return this;
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
		oneeSama.dice = update.dice;
		oneeSama.links = update.links;
		oneeSama.callback = this.inject;
		oneeSama.$buffer = this.$blockquote;
		oneeSama.state = update.state;
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
		let el = this.el.getElementsByTagName('time')[0];
		el.outerHTML = oneeSama.time(this.model.get('time'));
	},
	renderBacklinks(model, links) {
		// No more backlinks, because posts deleted or something
		if (!links && this.backlinks) {
			main.follow(() => this.backlinks.innerHTML = '');
			this.backlinks = null;
			return;
		}
		if (!this.backlinks)
			this.backlinks = this.el.getElementsByTagName('small')[0];
		let html = 'Replies:';
		const thread = state.page.get('thread'),
			notBoard = thread !== 0;
		for (var key in links) {
			// points to a different thread from the current
			const diff = links[key] !== thread;
			html += common.parseHTML
				` <a class="history" href="${diff && links[key]}#${key}">
					&gt;&gt;${key}${diff && notBoard && ' →'}
				</a>`;
		}
		main.follow(() => this.backlinks.innerHTML = html);
	},
	renderMenu(e) {
		new Menu({
			parent: e.target,
			model: this.model
		});
	},
	// Admin JS injections
	fun() {
		// Fun goes here
	},
	// Self-delusion tripfag filter
	anonymise() {
		this.el
			.getElementsByClassName('name')[0]
			.innerHTML = `<b class="name">${lang.anon}<b>`;
	},
	// Restore regular name
	renderName() {
		this.el
			.getElementsByClassName('name')[0]
			.outerHTML = oneeSama.name(this.model.attributes);
	},
	quotePost(e) {
		e.preventDefault();

		// TODO: Set highlighted post

		/*
		 Make sure the selection both starts and ends in the quoted post's
		 blockquote
		 */
		var sel,
			$post = this.$el,
			gsel = getSelection();
		const num = this.model.get('num');

		function isInside(p) {
			var $el = $(gsel[p] && gsel[p].parentElement);
			return $el.closest('blockquote').length
				&& $el.closest('article, section').is($post);
		}

		if (isInside('baseNode') && isInside('focusNode'))
			sel = gsel.toString();
		main.follow(function() {
			main.command('openPostBox', num);
			main.request('postForm').addReference(num, sel);
		});
	}
};

_.extend(module.exports, imager.Hidamari);
