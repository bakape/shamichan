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

	// Shift OP's replies on board pages
	if (op && !state.page.get('thread'))
		state.posts.get(op).dispatch('shiftReplies', true);

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

// TODO: Unify self-updates with OneeSama; this is redundant
main.oneeSama.hook('insertOwnPost', ({links}) => {
	if (!postForm || !links)
		return;
	postForm.$buffer.find('.nope').each(function () {
		var $a = $(this);
		const text = $a.text(),
			m = text.match(/^>>(\d+)/);
		if (!m)
			return;
		const num = m[1],
			op = links[num];
		if (!op)
			return;
		let $ref = $(common.join([postForm.imouto.postRef(num, op, false)]));
		$a.attr('href', $ref.attr('href')).attr('class', 'history');
		const refText = $ref.text();
		if (refText != text)
			$a.text(refText);
	});
});

const ArticleComposer = Article.extend({
	events: {
		'click #cancel': 'cancel',
		'change #imageInput': 'onImageChosen',
		'input #trans': 'onInput',
		'keydown #trans': 'onKeyDown',
		'click #done': 'finish',
		'click #toggle': 'onToggle'
	},
	initialize(args) {
		// super() call
		Article.prototype.initialize.call(this);
		this.listenTo(this.model, {
			'change': this.renderButtons,
			'change:spoiler': this.renderSpoilerPane
		});
		this.render(args).insertIntoDOM(args);
		this.model.set({
			spoiler: 0,
			nextSpoiler: -1
		});

		this.pending = '';
		this.line_count = 1;
		this.char_count = 0;

		// Initialize the form's private rendering singleton instance
		const imouto = this.imouto = new common.OneeSama({
			callback: inject,
			op: state.page.get('thread'),
			blockqoute: this.blockqoute,
			eLinkify: main.oneeSama.eLinkify,
			lang: main.lang,
			tamashii(num) {
				let section = document.query('#p' + num);
				section = section && section.closest('section');
				if (section) {
					const desc = num in state.mine.readAll() && lang.you;
					return this.postRef(num, util.getNum(section), desc);
				}
				else
					return `<a class="nope">&gt;&gt;${num}</a>`;
			}
		});
		imouto.setModel(this.model);
	},
	// Initial render
	render() {
		// Define attributes separatly for readbility
		const attrs = {
			input: {
				name: 'body',
				id: 'input',
				rows: 1,
				class: 'themed exclude',
				autocomplete: main.isMobile
			},
			form: {
				method: 'post',
				enctype: 'multipart/form-data',
				target: 'upload',
				id: 'uploadForm'
			},
			cancel: {
				type: 'buttom',
				value: lang.cancel,
				id: 'cancel'
			},
			imageInput: {
				type:'file',
				id: 'imageInput',
				name: 'image',
				accept: 'image/*;.webm;.pdf;.mp3'
			},
			toggle: {
				type: 'button',
				id: 'toggle'
			}
		};

		this.el.innerHTML = parseHTML
			`<header>
				<a class="name nope" target="_blank">
					<b></b>
				</a>
			</header>
			<span class="oi control" data-glyph="chevron-bottom"></span>
			<div class="container">
				<blockquote>
					<p id="buffer" class="exclude"></p>
					<p id="lineBuffer" class="exclude"></p>
					<textarea ${attrs.input}></textarea>
				</blockquote>
				<form ${attrs.form}>
					<input ${attrs.cancel}>
					<input ${attrs.imageInput}>
					<input ${attrs.toggle}>
					<strong id="uploadStatus"></strong>
				</form>
				<small></small>
			</div>`;

		// Cache elements to avoid lookup
		const els = ['buffer', 'lineBuffer', 'input', 'uploadForm', 'cancel',
			'imageInput', 'toggle', 'uploadStatus', 'hiddenUpload', 'sizer'];
		for (let el of els) {
			this[el] = document.query('#' + el);
		}
		this.renderIdentity();
		return this;
	},
	// Name, emaail and tripcode field
	renderIdentity() {
		// Model has already been alocated and has a proper identity rendered
		if (this.model.get('num'))
			return;
		const parsed = common.parse_name(main.$name.val(), main.$email.val()),
			haveTrip = !!(parsed[1] || parsed[2]);
		const el = this.el.query('.name'),
			name = el.query('a');
		if (parsed[0])
			name.textContent = parsed[0] + ' ';
		else
			name.tetxContent = haveTrip ? '' : lang.anon;
		if (haveTrip)
			util.parseDOM(' <code>!?</code>').forEach(el => name.after(el));

		// Insert staff title
		main.oneeSama.trigger('fillMyName', name);
		const email = main.$email.val().trim();
		if (email) {
			el.setAttribute('href', 'mailto:' + email);
			el.classList.remove('nope');
			el.classList.add('email');
		}
		else {
			el.removeAttribute('href');
			el.classList.add('nope');
			el.classList.remove('email');
		}
	},
	insertIntoDOM({destination}) {
		destination.before(this.el);
		this.resizeInput();
		main.$threads.queryAll('aside.postsing').forEach(el =>
			el.style.display = 'none');
		this.fun();
	},
	/*
	 Allows keeping the input buffer sized as if the text was monospace,
	 without actually displaying monospace font. Keeps the input buffer from
	 shifting around needlessly.
	 */
	resizeInput(val) {
		const {sizer, input} = this;
		if (typeof val !== 'string')
			val = input.value;
		sizer.textContent = val;
		let size = sizer.width + common.INPUT_ROOM;
		size = Math.max(size, inputMinSize
			- input.getBoundingClientRect().left
			- this.el.getBoundingClientRect().left);
		this.input.style.width = size + 'px';
	},
	renderButtons() {
		const {num, uploading, uploaded, uploadStatus, sentAllocRequest}
			= this.model.attributes;
		const allocWait = sentAllocRequest && !num;
		this.submit.disabled = !!(uploading || allocWait);
		if (uploaded)
			this.submit.style.marginLeft = 0;
		this.cancel.disabled = !!allocWait;
		this.cancel.style.display = (!num || uploading) ? '' : 'none';
		this.imageInput.disabled = !!uploading;
		this.uploadStatus.innerHTML = uploadStatus;
	},
	renderSpoilerPane(model, spoiler) {
		const background = spoiler
			? `${config.MEDIA_URL}spoil/spoil${spoiler}.png`
			: config.MEDIA_URL + 'css/ui/pane.png';
		this.toggle.style.backgroundImage = `url("${background}")`;
	}
});

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

		// Initialize the form's private rendering singleton instance
		let imouto = this.imouto = new common.OneeSama({
			callback: inject,
			op: state.page.get('thread'),
			state: [common.S_BOL, 0],

			// TODO: Convert current OneeSama.state array to more flexible
			// object
			state2: {spoiler: 0},
			$buffer: this.$buffer,
			eLinkify: main.oneeSama.eLinkify,
			lang: main.lang,
			tamashii(num) {
				let section = document.query('#p' + num);
				section = section && section.closest('section');
				if (section) {
					const desc = num in state.mine.readAll() && this.lang.you;
					return this.postRef(num, util.getNum(section), desc);
				}
				else
					return `<a class="nope">&gt;&gt;${num}</a>`;
			}
		});
		imouto.hook('spoilerTag', util.touchable_spoiler_tag);
		main.oneeSama.trigger('imouto', imouto);
	},
	// Initial render
	render({destination, section}) {
		const op = this.model.get('op');
		this.setElement((op ? document.createElement('article') : section));

		// A defined op means the post is a reply, not a new thread
		this.isThread = !op;

		this.$buffer = $('<p/>');
		this.$lineBuffer = $('<p/>');
		this.$meta = $('<header><a class="nope"><b/></a> <time/></header>');
		this.$input = $('<textarea/>', {
			name: 'body',
			id: 'trans',
			rows: '1',
			class: 'themed',
			autocomplete: main.isMobile
		});
		this.$submit = $('<input/>', {
			id: 'done',
			type: 'button',
			value: main.lang.done
		});
		this.$subject = $('<input/>', {
			id: 'subject',
			class: 'themed',
			maxlength: state.hotConfig.SUBJECT_MAX_LENGTH,
			width: '80%'
		});
		this.$blockquote = $('<blockquote/>');

		/*
		 Allows keeping the input buffer sized as if the text was monospace,
		 without actually displaying monospace font. Keeps the input buffer from
		 shifting around needlessly.
		 */
		this.$sizer = $('<pre/>').appendTo('body');

		this.$blockquote.append(this.$buffer, this.$lineBuffer, this.$input);
		this.$el.append(this.$meta, this.$blockquote, '<small/>');
		if (this.isThread) {
			this.$el.append(`<label for="subject">${lang.subject}: </label>`,
				this.$subject);
			this.$blockquote.hide();
		}
		this.$uploadForm = this.renderUploadForm();
		this.$el.append(this.$uploadForm);

		// Add a menu to the postform
		main.oneeSama.trigger('draft', this.$el);
		this.renderIdentity();

		// Insert into the DOM
		destination.style.display = 'none';
		if (this.isThread) {
			destination.after(this.el);
			this.el.after(document.createElement('hr'));
			this.$subject.focus();
		}
		else {
			destination.before(this.el);
			this.resizeInput();
			this.$input.focus();
		}

		main.$threads.find('aside.posting').hide();
		this.fun();
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
	renderButtons() {
		const attrs = this.model.attributes,
			allocWait = attrs.sentAllocRequest && !attrs.num,
			d = attrs.uploading || allocWait;
		// Beware of undefined!
		this.$submit.prop('disabled', !!d);
		if (attrs.uploaded)
			this.$submit.css({'margin-left': '0'});
		this.$cancel.prop('disabled', !!allocWait);
		this.$cancel.toggle(!!(!attrs.num || attrs.uploading));
		this.$imageInput.prop('disabled', !!attrs.uploading);
		this.$uploadStatus.html(attrs.uploadStatus);
	},
	renderSpoilerPane(model, sp) {
		const background = sp ? `${config.MEDIA_URL}spoil/spoil${sp}.png`
			: config.MEDIA_URL + 'css/ui/pane.png';
		this.$toggle.css('background-image', `url("${background}")`);
	},
	renderUploadForm() {
		var $form = $('<form method="post" enctype="multipart/form-data" '
			+ 'target="upload"></form>');
		this.$cancel = $('<input/>', {
			type: 'button',
			value: lang.cancel,
			click: $.proxy(this, 'cancel')
		});
		this.$imageInput = $('<input/>', {
			type: 'file',
			id: 'image',
			name: 'image',
			accept: config.WEBM ? 'imager/*;.webm' : 'image/*',
			change: $.proxy(this, 'onImageChosen')
		});
		this.$toggle = $('<input/>', {
			type: 'button',
			id: 'toggle'
		});
		this.$uploadStatus = $('<strong/>');
		$form.append(this.$cancel,
			this.$imageInput,
			this.$toggle, ' ',
			this.$uploadStatus);
		this.$iframe = $('<iframe/>', {
			src: '',
			name: 'upload',
			id: 'hidden-upload'
		}).appendTo('body');
		this.model.set({
			spoiler: 0,
			nextSpoiler: -1
		});
		return $form;
	},
	// Cancel file upload
	cancel() {
		if (this.model.get('uploading')) {
			this.$iframe.remove();
			this.$iframe = $('<iframe></iframe>', {
				src: '',
				name: 'upload',
				id: 'hidden-upload'
			}).appendTo('body');
			this.uploadError('');
			this.model.set({cancelled: true});
		}
		else
			this.finish();
	},
	onImageChosen() {
		if (this.model.get('uploading') || this.model.get('uploaded'))
			return;
		if (!this.$imageInput.val()) {
			this.model.set('uploadStatus', '');
			return;
		}
		const extra = this.prepareUpload();
		for (var k in extra) {
			$('<input type=hidden>')
				.attr('name', k)
				.val(extra[k])
				.appendTo(this.$uploadForm);
		}
		this.$uploadForm.prop('action', util.uploadURL());
		this.$uploadForm.submit();
		this.$iframe.load(function() {
			if (!postForm)
				return;
			var doc = this.contentWindow || this.contentDocument;
			if (!doc)
				return;
			try {
				var error = $(doc.document || doc).text();
				/*
				 if it's a real response, it'll postMessage to us, so we don't have
				 to do anything.
				 */
				if (/legitimate imager response/.test(error))
					return;
				// sanity check for weird browser responses
				if (error.length < 5 || error.length > 100)
					error = lang.unknownUpload;
				postForm.uploadError(error);
			}
			catch(e) {
				/*
				 likely cross-origin restriction
				 wait before erroring in case the message shows up
				 */
				setTimeout(function() {
					postForm.uploadFallbackMessage();
				}, 500);
			}
		});
		this.notifyUploading();
	},
	prepareUpload() {
		this.model.set('uploadStatus', lang.uploading);
		this.$input.focus();
		const attrs = this.model.attributes;
		return {spoiler: attrs.spoiler, op: attrs.op || 0};
	},
	/*
	 this is just a fallback message for when we can't tell, if there was an
	 error due to cross-origin restrictions
	 */
	uploadFallbackMessage() {
		var a = this.model.attributes,
			stat = a.uploadStatus;
		if (!a.cancelled && a.uploading && (!stat || stat == lang.uploading))
			this.model.set('uploadStatus', lang.unknownResult);
	},
	notifyUploading() {
		this.model.set({uploading: true, cancelled: false});
		this.$input.focus();
	},
	resizeInput(val) {
		if (typeof val !== 'string')
			val = this.$input.val();
		this.$sizer.text(val);
		var size = this.$sizer.width() + common.INPUT_ROOM;
		size = Math.max(size, inputMinSize
			- this.$input.offset().left - this.$el.offset().left);
		this.$input.css('width', size + 'px');
	},
	onInput(val) {
		if (val === undefined || val instanceof $.Event)
			val = this.$input.val();
		var start = this.$input[0].selectionStart,
			end = this.$input[0].selectionEnd;

		var changed = false,
			m, time, video;

		// Turn YouTube links into proper refs
		while(true) {
			m = val.match(embed.youtube_url_re);
			if (!m)
				break;
			// Substitute
			time = this.findTimeArg(m[3])
				|| this.findTimeArg(m[1])
				|| m[4]
				|| '';
			video = '>>>/watch?v=' + m[2] + time;
			val = embedRewrite(m, video);
		}

		//Youtu.be links
		while(true) {
			m = val.match(embed.youtube_short_re);
			if (!m)
				break;
			// Substitute
			time = this.findTimeArg(m[2]) || '';
			video = '>>>/watch?v=' + m[1] + time;
			val = embedRewrite(m, video);
		}

		// SoundCloud links
		while(true) {
			m = val.match(embed.soundcloud_url_re);
			if (!m)
				break;
			var sc = '>>>/soundcloud/' + m[1];
			val = embedRewrite(m, sc);
		}

		// Pastebin links
		while(true) {
			m = val.match(embed.pastebin_re);
			if (!m)
				break;
			var pbin = '>>>/pastebin/' + m[1];
			val = embedRewrite(m, pbin);
		}

		// Rewite embedable URLs to native embed URL syntax
		function embedRewrite(m, rw) {
			var old = m[0].length;
			var newVal = val.substr(0, m.index) + rw + val.substr(m.index + old);
			changed = true;
			if (m.index < start) {
				var diff = old - rw.length;
				start -= diff;
				end -= diff;
			}
			return newVal;
		}

		if (changed)
			this.$input.val(val);

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
	uploadError(msg) {
		if (this.model.get('cancelled'))
			return;
		this.model.set({
			uploadStatus: msg,
			uploading: false
		});
		if (this.$uploadForm)
			this.$uploadForm.find('input[name=alloc]').remove();
	},
	uploadStatus(msg) {
		if (this.model.get('cancelled'))
			return;
		this.model.set('uploadStatus', msg);
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
	// Extend with imager.js methods
	renderImage: imager.Hidamari.renderImage,
	// Overrides automatic image expansion, if any
	autoExpandImage() {
		return this;
	},
	fun() {
		
	}
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
