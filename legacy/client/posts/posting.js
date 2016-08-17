/*
 * Evertything related to writing and commiting posts
 */

const Article = require('./article'),
	main = require('../main'),
	embed = require('./embed'),
	ident = require('./identity'),
	imager = require('./imager'),
	{$, _, Backbone, Cookie, common, config, connSM, util, lang, oneeSama,
		options, postSM, state} = main;

let postForm, postModel;
/*
 The variable gets overwritten, so a simple refference will not do. Calling a
 fucntion to retrieve the var each time solves the problem.
 */
main.reply('postForm', () => postForm)
	.reply('postModel', () => postModel)
	.reply('postForm:indentity', () => postForm && postForm.renderIdentity());

// Minimal size of the input buffer
let	inputMinSize = 300;
// For mobile
if (window.screen && screen.width <= 320)
	inputMinSize = 50;

const ComposerModel = Backbone.Model.extend({idAttribute: 'num'});

// Synchronyse postform state with websocket connectivity
connSM.on('synced', postSM.feeder('sync'));
connSM.on('dropped', postSM.feeder('desync'));
connSM.on('desynced', postSM.feeder('desync'));

// Allow remotely altering posting FSM state
main.reply('postSM:feed', state => postSM.feed(state));

postSM.act('* + desync -> none', () => {
	// TODO: Desync logic
	if (postForm) {
		postForm.el.classList.remove('editing');
		postForm.$input.val('');
		postForm.finish();
	}
	main.$threads.find('aside.posting').hide();
});

postSM.act('none + sync, draft, alloc + done -> ready', () => {
	// TODO: Add unfinished post checking

	if (postForm) {
		postForm.remove();
		postForm = postModel = null;
	}
	for (let el of main.$threads[0].queryAll('aside.posting')) {
		el.style.display = '';
	}
});

// Make new postform
postSM.act('ready + new -> draft', aside => {
	let op,
		section = aside.closest('section');
	if (section)
		op = util.getNum(section);
	else
		section = document.createElement('section');

	postForm = new ComposerView({
		model: postModel = new ComposerModel({op}),
		destination: aside,
		section
	});
});

postSM.preflight('draft', aside => aside.matches('aside'));

postSM.act('draft + alloc -> alloc', msg => postForm.onAllocation(msg));

// Render image upload status messages
main.dispatcher[common.IMAGE_STATUS] = msg =>
	postForm && postForm.dispatch(msg[0]);

main.$doc.on('click', 'aside.posting a', function () {
	postSM.feed('new', this.parentNode);
});

main.$doc.on('keydown', handle_shortcut);

function handle_shortcut(event) {
	if (!event.altKey)
		return;

	const opts = options.attributes;
	switch(event.which) {
		case opts.new:
			const aside = document.query('aside.posting');
			if (aside) {
				postSM.feed('new', aside);
				prevent();
			}
			break;
		case opts.togglespoiler:
			if (postForm) {
				postForm.onToggle(event);
				prevent();
			}
			break;
		case opts.done:
			if (postForm && !postForm.$submit.attr('disabled')) {
				postForm.finish();
				prevent();
			}
			break;
		// Insert text spoiler
		case opts.textSpoiler:
			if (postForm) {
				var postState = this.imouto.state2.spoiler;
				// Was spoiler already started?
				var sp = (postState ? '[/' : ' [') + 'spoiler]';
				this.imouto.state2.spoiler = !postState;
				this.$input.val(this.$input.val() + sp);
				prevent();
			}
			break;
		case opts.expandAll:
			imager.massExpander.toggle();
			prevent();
			break;
		case opts.workMode:
			const val = main.oneeSama.workMode = !main.oneeSama.workMode;
			Cookie.set('workModeTOG', val);
			const banner = document.querySelector("h1 > img");
			if(banner!=null)
				banner.style.display =  val? 'none':'';
			document.getElementById('theme').setAttribute('href',
					`${config.MEDIA_URL}css/${val? state.hotConfig.get('DEFAULT_CSS'): main.options.get("theme")}.css?v=${main.cssHash}`);
			oneeSama.thumbStyle = val? 'hide': main.options.get('thumbs');
			main.options.trigger("workModeTOG");
			window.addEventListener('beforeunload', function () {
				Cookie.set("workModeTOG",false);
			});
			prevent()
			break;
	}

	function prevent() {
		event.stopImmediatePropagation();
		event.preventDefault();
	}
}

const ComposerView = Backbone.View.extend({
	events: {
		'input #trans': 'onInput',
		'keydown #trans': 'onKeyDown',
		'click #done': 'finish',
		'click #toggle': 'onToggle'
	},

	initialize(args) {
		this.listenTo(this.model, {
			'change': this.renderButtons,
			'change:spoiler': this.renderSpoilerPane
		});

		this.render(args);

		this.pending = '';
		this.line_count = 1;
		this.char_count = 0;

		imouto.hook('spoilerTag', util.touchable_spoiler_tag);
		main.oneeSama.trigger('imouto', imouto);
	},

	// Initial render
	render({destination, section}) {
		this.$meta = $('<header><a class="nope"><b/></a> <time/></header>');

		// Add a menu to the postform
		main.oneeSama.trigger('draft', this.$el);
		this.renderIdentity();

		main.$threads.find('aside.posting').hide();
	},

	// Render the name, email, and admin title, if any
	renderIdentity() {
		// Model has already been alocated and has a proper identity rendered
		if (this.model.get('num'))
			return;
		const parsed = common.parse_name(main.$name.val(), main.$email.val()),
			haveTrip = !!(parsed[1] || parsed[2]);
		let $b = this.$meta.find('b');
		if (parsed[0])
			$b.text(parsed[0] + ' ');
		else
			$b.text(haveTrip ? '' : main.lang.anon);
		if (haveTrip)
			$b.append(' <code>!?</code>');

		// Insert staff title
		main.oneeSama.trigger('fillMyName', $b);
		const email = main.$email.val().trim();
		let $tag = this.$meta.children('a').first();
		if (email) {
			$tag.attr({
				href: 'mailto:' + email,
				target: '_blank',
				class: 'email'
			});
		}
		else
			$tag.removeAttr('href').removeAttr('target').attr('class', 'nope');
	},

	onInput(val) {
		if (val === undefined || val instanceof $.Event)
			val = this.$input.val();
		var start = this.$input[0].selectionStart,
			end = this.$input[0].selectionEnd;

		var nl = val.lastIndexOf('\n');
		if (nl >= 0) {
			var ok = val.substr(0, nl);
			val = val.substr(nl + 1);
			this.$input.val(val);
			if (this.model.get('sentAllocRequest') || /[^ ]/.test(ok))
				this.commit(ok + '\n');
		}
		else {
			m = val
				.split('')
				.reverse()
				.join('')
				.match(/^(\s*\S+\s+\S+)\s+(?=\S)/);
			if (m) {
				var lim = val.length - m[1].length;
				this.commit(val.substr(0, lim));
				val = val.substr(lim);
				start -= lim;
				end -= lim;
				this.$input.val(val);
				this.$input[0].setSelectionRange(start, end);
			}
		}

		this.$input.attr('maxlength', common.MAX_POST_CHARS - this.char_count);
		this.resizeInput(val);
	},

	findTimeArg(params) {
		if (!params || params.indexOf('t=') < 0)
			return false;
		params = params.split('&');
		for (let i = 0, len = params.length; i < len; i++) {
			let pair = '#p' + params[i];
			if (embed.youtube_time_re.test(pair))
				return pair;
		}
		return false;
	},

	// Commit any staged words to the server
	commit(text) {
		var lines;
		if (text.indexOf('\n') >= 0) {
			lines = text.split('\n');
			this.line_count += lines.length - 1;
			const breach = this.line_count - common.MAX_POST_LINES + 1;
			if (breach > 0) {
				for (var i = 0; i < breach; i++)
					lines.pop();
				text = lines.join('\n');
				this.line_count = common.MAX_POST_LINES;
			}
		}
		const left = common.MAX_POST_CHARS - this.char_count;
		if (left < text.length)
			text = text.substr(0, left);
		if (!text)
			return;
		this.char_count += text.length;

		// Either get an allocation or send the committed text
		const attrs = this.model.attributes;
		if (!attrs.num && !attrs.sentAllocRequest) {
			main.send([common.INSERT_POST, this.allocationMessage(text, null)]);
			this.model.set({sentAllocRequest: true});
		}
		else if (attrs.num)
			main.send(text);
		else
			this.pending += text;

		// Add it to the user's display
		if (lines) {
			lines[0] = this.$lineBuffer.text() + lines[0];
			this.$lineBuffer.text(lines.pop());
			for (let o = 0, len = lines.length; o < len; o++)
				this.imouto.fragment(lines[o] + '\n');
		}
		else {
			this.$lineBuffer.append(document.createTextNode(text));
			this.$lineBuffer[0].normalize();
		}
	},

	// Construct the message for post allocation in the database
	allocationMessage(text, image) {
		var msg = {nonce: main.request('nonce:create')};

		function opt(key, val) {
			if (val)
				msg[key] = val;
		}

		opt('name', main.$name.val().trim());
		opt('email', main.$email.val().trim());
		opt('subject', this.$subject.val().trim());
		opt('frag', text);
		opt('image', image);
		opt('op', this.model.get('op'));

		return msg;
	},

	onKeyDown(event) {
		main.follow(() => {
			switch(event.which) {
				case 13:
					event.preventDefault();
				// fall-through
				case 32:
					// predict result
					var input = this.$input[0];
					var val = this.$input.val();
					val = val.slice(0, input.selectionStart)
						+ (event.which == 13 ? '\n' : ' ')
						+ val.slice(input.selectionEnd);
					this.onInput(val);
					break;
				default:
					handle_shortcut.bind(this)(event);
			}
		});
	},

	finish() {
		if (this.model.get('num')) {
			this.flushPending();
			this.commit(this.$input.val());
			this.$input.remove();
			this.$submit.remove();
			if (this.$uploadForm)
				this.$uploadForm.remove();
			if (this.$iframe) {
				this.$iframe.remove();
				this.$iframe = null;
			}
			this.imouto.fragment(this.$lineBuffer.text());
			this.$buffer.replaceWith(this.$buffer.contents());
			this.$lineBuffer.remove();
			this.$blockquote.css({
				'margin-left': '', 'padding-left': ''
			}
			);
			main.send([common.FINISH_POST]);
			this.preserve = true;
			if (this.isThread)
				this.$el.append(main.oneeSama.replyBox());

			let missing = this.imouto.allRolls.sent - this.imouto.allRolls.seen;
			//if missing>0 we have to wait until insertOwnPosts "sees" the
			// dice
			if (missing > 0) {
				let checkAgain;
				(checkAgain= (n) => {
					setTimeout(()=> {
						if (this.imouto.allRolls.seen == this.imouto.allRolls.sent || n ==0)
							postSM.feed('done');
						else
							checkAgain(n - 1);
					}, 100);
				})(5); //retry 5 times
			}
			else
				postSM.feed('done');
		}else
			postSM.feed('done');
	},

	// Send any unstaged words
	flushPending() {
		if (this.pending) {
			main.send(this.pending);
			this.pending = '';
		}
	},

	onToggle(event) {
		const attrs = this.model.attributes;
		if (attrs.uploading || attrs.uploaded)
			return;
		event.preventDefault();
		event.stopImmediatePropagation();
		if (attrs.spoiler) {
			this.model.set({spoiler: 0});
			return;
		}
		const pick = common.pick_spoiler(attrs.nextSpoiler);
		this.model.set({
			spoiler: pick.index,
			nextSpoiler: pick.next
		});
	},

	onAllocation(msg) {
		const num = msg.num;
		state.ownPosts[num] = num;
		this.model.set({num: num});
		this.flushPending();
		var header = $(main.oneeSama.header(msg));
		this.$meta.replaceWith(header);
		this.$meta = header;
		if (!this.isThread)
			this.$el.addClass('editing');

		/*
		 TODO: Hide threads that are over THREADS_PER_PAGE. Also would need to be
		 removed from syncs client and server-side. Hmm.
		 */

		this.$el.attr('id', 'p' + num);

		if (msg.image)
			this.insertUploaded(msg.image);

		if (this.$uploadForm)
			this.$uploadForm.append(this.$submit);
		else
			this.$blockquote.after(this.$submit);
		if (this.isThread) {
			this.$subject.siblings('label').andSelf().remove();
			this.$blockquote.show();
			this.resizeInput();
			this.$input.focus();
		}

		/*
		 Ensures you are nagged at by the browser, when navigating away from an
		 unfinished allocated post.
		 */
		window.onbeforeunload = function() {
			return "You have an unfinished post.";
		};
	},

	// Insert an image that has been uploaded and processed by the server
	insertUploaded(info) {
		this.renderImage(null, info);
		this.$imageInput
			.siblings('strong')
			.andSelf()
			.remove();
		this.$cancel.remove();
		this.$uploadForm.find('#toggle').remove();
		this.flushPending();
		this.model.set({
			uploading: false,
			uploaded: true,
			sentAllocRequest: true
		});

		// Stop obnoxious wrap-around-image behaviour
		var $img = this.$el.find('img');
		this.$blockquote.css({
			'margin-left': $img.css('margin-right'),
			'padding-left': $img.width()
		});

		this.resizeInput();
	},

	// Handle image upload status
	dispatch(msg) {
		const a = msg.arg;
		switch(msg.t) {
			case 'alloc':
				this.onImageAllocation(a);
				break;
			case 'error':
				this.uploadError(a);
				break;
			case 'status':
				this.uploadStatus(a);
				break;
		}
	},

	onImageAllocation(msg) {
		const attrs = this.model.attributes;
		if (attrs.cancelled)
			return;
		if (!attrs.num && !attrs.sentAllocRequest) {
			main.send([common.INSERT_POST, this.allocationMessage(null, msg)]);
			this.model.set({sentAllocRequest: true});
		}
		else
			main.send([common.INSERT_IMAGE, msg]);
	},

	addReference(num, sel) {
		// If a >>link exists, put this one on the next line
		var val = this.$input.val();
		if (/^>>\d+$/.test(val)) {
			this.$input.val(val + '\n');
			this.onInput();
			val = this.$input.val();
		}
		// Quote selected text automatically
		if (sel) {
			sel = sel.split('\n');
			// Prepend > to each line
			for (let i = 0, len = sel.length; i < len; i++)
				sel[i] = '>' + sel[i];
			num += '\n' + sel.join('\n') + '\n';
		}
		this.$input.val(val + '>>' + num);
		this.$input[0].selectionStart = this.$input.val().length;
		this.onInput();
		this.$input.focus();
	},

	remove() {
		if (!this.preserve) {
			if (this.isThread)
				this.$el.next('hr').remove();
			this.$el.remove();
		}
		this.$sizer.remove();
		if (this.$iframe) {
			this.$iframe.remove();
			this.$iframe = null;
		}
		this.stopListening();
		window.onbeforeunload = null;
	},
});
exports.ComposerView = ComposerView;

function openPostBox(num) {
	postSM.feed('new', document.query(`#p${num} aside.posting`));
}
main.reply('openPostBox', openPostBox);

window.addEventListener('message', function(event) {
	const msg = event.data;
	if (msg !== 'OK' && postForm)
		postForm.uploadError(msg);
}, false);

main.$threads.on('click', 'a.quote', function(e) {
	e.preventDefault();

	// TODO: Set highlighted post

	/*
	 Make sure the selection both starts and ends in the quoted post's
	 blockquote
	 */
	const post = e.target.closest('article, section'),
		gsel = getSelection(),
		num = util.getNum(post);

	function isInside(prop) {
		const el = gsel[prop] && gsel[prop].parentElement;
		return  el
			&& el.closest('blockquote')
			&& el.closest('article, section') === post;
	}

	let sel;
	if (isInside('baseNode') && isInside('focusNode'))
		sel = gsel.toString();
	openPostBox(util.getNum(post.closest('section')));
	postForm.addReference(num, sel);
});
