/*
 * Evertything related to writing and commiting posts
 */
'use strict';

var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	client = require('../client'),
	common = require('../../common'),
	embed = require('./embed'),
	ident = require('./identity'),
	imager = require('./imager'),
	inject = require('./common').inject,
	main = require('../main'),
	nonce = require('./nonce'),
	options = require('../options'),
        scroll = require('../scroll'),
	state = require('../state'),
            thread = state.page.get('thread');
    
var connSM = main.connSM,
	postSM = main.postSM;
const uploadingMessage = 'Uploading...';
var postForm = main.postForm,
// Minimal size of the input buffer
	inputMinSize = 300;

// For mobile
if (window.screen && screen.width <= 320)
	inputMinSize = 50;

var ComposerModel = Backbone.Model.extend({
	idAttribute: 'num'
});

// Synchronyse postform state with websocket connectivity
connSM.on('synced', postSM.feeder('sync'));
connSM.on('dropped', postSM.feeder('desync'));
connSM.on('desynced', postSM.feeder('desync'));

postSM.act('* + desync -> none', function() {
	// TODO: Desync logic
	if (postForm) {
		postForm.$el.removeClass('editing');
		postForm.$input.val('');
		postForm.finish();
	}
	main.$threads.find('aside').hide();
});

postSM.act('none + sync, draft, alloc + done -> ready', function() {
	// TODO: Add unfinished post checking

	if (postForm) {
		postForm.remove();
		main.postForm = postForm = null;
		main.postModel = null;
	}
	main.$threads.find('aside').show();
});

// Make new postform
postSM.act('ready + new -> draft', function($aside) {
	var op = null;
	var $sec = $aside.closest('section');
	if ($sec.length)
		op = extractNum($sec);
	else
		$sec = $('<section/>');

	// Shift OP's replies on board pages
	if (op)
		state.posts.get(op).trigger('shiftReplies', true);

	main.postModel = new ComposerModel({op: op});
	main.postForm = postForm = new ComposerView({
		model: main.postModel,
		$dest: $aside,
		$sec: $sec
	});
});

// Extract post number
function extractNum($post) {
	return parseInt($post.attr('id'), 10);
}

postSM.preflight('draft', function(aside) {
	return aside.is('aside');
});

postSM.act('draft + alloc -> alloc', function(msg) {
	postForm.onAllocation(msg);
});

// Render image upload status messages
main.dispatcher[common.IMAGE_STATUS] = function(msg) {
	if (postForm)
		postForm.dispatch(msg[0]);
};

main.$doc.on('click', 'aside a', _.wrap(function () {
	postSM.feed('new', $(this).parent());
}, scroll.followLock));

main.$doc.on('keydown', handle_shortcut);

function handle_shortcut(event) {
	if (!event.altKey)
		return;

	var used = false,
		opts = options.attributes;
	switch(event.which) {
		case opts.new:
			var $aside = state.page.get('thread') ? main.$threads.find('aside')
				: $ceiling().next();
			if ($aside.is('aside') && $aside.length === 1) {
				scroll.followLock(function() {
                                    postSM.feed('new', $aside);
                                });
				used = true;
			}
			break;
		case opts.togglespoiler:
			if (postForm) {
				postForm.onToggle(event);
				used = true;
			}
			break;
		case opts.done:
			if (postForm && !postForm.$submit.attr('disabled')) {
					postForm.finish_wrapped();
					used = true;
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
				used = true;
			}
			break;
		case opts.expandAll:
			imager.massExpander.toggle();
			used = true;
			break;
	}

	if (used) {
		event.stopImmediatePropagation();
		event.preventDefault();
	}
}

// Gets the top <hr> of <threads>
function $ceiling() {
	return main.$threads.children('hr').first();
}

// TODO: Unify self-updates with OneeSama; this is redundant
main.oneeSama.hook('insertOwnPost', function (info) {
	if (!main.postForm || !info.links)
		return;
	postForm.$buffer.find('.nope').each(function() {
		var $a = $(this);
		const text = $a.text(),
			m = text.match(/^>>(\d+)/);
		if (!m)
			return;
		const num = m[1],
			op = info.links[num];
		if (!op)
			return;
		var $ref = $(common.flatten(
				main.postForm.imouto.post_ref(num, op, false)
			).join('')
		);
		$a.attr('href', $ref.attr('href')).removeAttr('class');
		const refText = $ref.text();
		if (refText != text)
			$a.text(refText);
	});
});

var ComposerView = Backbone.View.extend({
	events: {
		'input #trans': 'onInput',
		'keydown #trans': 'onKeyDown',
		'click #done': 'finish_wrapped',
		'click #toggle': 'onToggle'
	},

	initialize: function(args) {
		this.listenTo(this.model, {
			'change': this.renderButtons,
			'change:spoiler': this.renderSpoilerPane
		});

		this.render(args);

		this.pending = '';
		this.line_count = 1;
		this.char_count = 0;

		// The form's own dedicated renderer instance
		this.imouto = new common.OneeSama(function(num) {
			var $sec = $('#' + num);
			if (!$sec.is('section'))
				$sec = $sec.closest('section');
			if ($sec.is('section'))
				this.callback(this.post_ref(num, extractNum($sec)));
			else
				this.callback(common.safe(`<a class="nope">&gt;&gt;${num}</a>`));
		});
		// Initialise the renderer instance
		this.imouto.callback = inject;
		this.imouto.op = state.page.get('thread');
		this.imouto.state = [common.S_BOL, 0];
		// TODO: Convert current OneeSama.state array to more flexible object
		this.imouto.state2 = {spoiler: 0};
		this.imouto.$buffer = this.$buffer;
		this.imouto.eLinkify = main.oneeSama.eLinkify;
		this.imouto.hook('spoilerTag', client.touchable_spoiler_tag);
		main.oneeSama.trigger('imouto', this.imouto);
	},

	// Initial render
	render: function(args) {
		const op = this.model.get('op');
		this.setElement((op ? $('<article/>') : args.$sec)[0]);
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
			value: 'Done'
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

		// TODO: Shift the parrent sections replies on board pages

		this.$blockquote.append(this.$buffer, this.$lineBuffer, this.$input);
		this.$el.append(this.$meta, this.$blockquote);
		if (this.isThread) {
			this.$el.append('<label for="subject">Subject: </label>',
				this.$subject);
			this.$blockquote.hide();
		}
		this.$uploadForm = this.renderUploadForm();
		this.$el.append(this.$uploadForm);
		// Add a menu to the postform
		main.oneeSama.trigger('draft', this.$el);
		this.renderIdentity();
		args.$dest.hide();

		if (this.isThread) {
			this.$el.insertAfter(args.$dest);
			this.$el.after('<hr>');
			this.$subject.focus();
		}
		else {
			this.$el.insertBefore(args.$dest);
			this.resizeInput();
			this.$input.focus();
		}

		main.$threads.find('aside').hide();
		preloadPanes();
	},

	// Render the name, email, and admin title, if any
	renderIdentity: function() {
		// Model has already been alocated and has a proper identity rendered
		if (this.model.get('num'))
			return;
		const parsed = common.parse_name(main.$name.val(), main.$email.val()),
			haveTrip = !!(parsed[1] || parsed[2]);
		var $b = this.$meta.find('b');
		if (parsed[0])
			$b.text(parsed[0] + ' ');
		else
			$b.text(haveTrip ? '' : main.lang.anon);
		if (haveTrip)
			$b.append(' <code>!?</code>');
		// Insert staff title
		main.oneeSama.trigger('fillMyName', $b);
		var email = main.$email.val().trim();
		if (common.is_noko(email))
			email = '';
		var $tag = this.$meta.children('a').first();
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

	renderButtons: function() {
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

	renderSpoilerPane: function(model, sp) {
		this.$toggle.css('background-image', 'url("'
			+ (sp ? spoilerPaneUrl(sp)
				: main.config.MEDIA_URL + 'css/ui/pane.png')
			+ '")');
	},

	renderUploadForm: function() {
		var $form = $('<form method="post" enctype="multipart/form-data" '
			+ 'target="upload"></form>');
		this.$cancel = $('<input/>', {
			type: 'button',
			value: 'Cancel',
			click: $.proxy(this, 'cancel')
		});
		this.$imageInput = $('<input/>', {
			type: 'file',
			id: 'image',
			name: 'image',
			accept: main.config.WEBM ? 'imager/*;.webm' : 'image/*',
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
	cancel: function() {
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
			this.finish_wrapped();
	},

	onImageChosen: function() {
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
		this.$uploadForm.prop('action', imageUploadURL());
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
					error = 'Unknown upload error.';
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

	prepareUpload: function() {
		this.model.set('uploadStatus', uploadingMessage);
		this.$input.focus();
		const attrs = this.model.attributes;
		return {spoiler: attrs.spoiler, op: attrs.op || 0};
	},

	/*
	 this is just a fallback message for when we can't tell, if there was an
	 error due to cross-origin restrictions
	 */
	uploadFallbackMessage: function() {
		var a = this.model.attributes,
			stat = a.uploadStatus;
		if (!a.cancelled && a.uploading && (!stat || stat == uploadingMessage))
			this.model.set('uploadStatus', 'Unknown result.');
	},

	notifyUploading: function() {
		this.model.set({uploading: true, cancelled: false});
		this.$input.focus();
	},

	resizeInput: function(val) {
		if (typeof val !== 'string')
			val = this.$input.val();
		this.$sizer.text(val);
		var size = this.$sizer.width() + common.INPUT_ROOM;
		size = Math.max(size, inputMinSize
			- this.$input.offset().left - this.$el.offset().left);
		this.$input.css('width', size + 'px');
	},

	onInput: function(val) {
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
			video = '>>>/watch?v=' + m[1] + t;
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

	findTimeArg: function(params) {
		if (!params || params.indexOf('t=') < 0)
			return false;
		params = params.split('&');
		for (let i = 0, len = params.length; i < len; i++) {
			let pair = '#' + params[i];
			if (embed.youtube_time_re.test(pair))
				return pair;
		}
		return false;
	},

	// Commit any staged words to the server
	commit: function(text) {
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
	allocationMessage: function(text, image) {
		var msg = {nonce: nonce.create()};

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

	onKeyDown: function(event) {

		// TODO: Scrolling and locking to bottom

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
	},

	finish: function() {
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
				'margin-left': '',
				'padding-left': ''
			});
			main.send([common.FINISH_POST]);
			this.preserve = true;
			if (this.isThread)
				this.$el.append(main.oneeSama.replyBox());
		}
		postSM.feed('done');
	},

	// Send any unstaged words
	flushPending: function() {
		if (this.pending) {
			main.send(this.pending);
			this.pending = '';
		}
	},

	onToggle: function(event) {
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

	onAllocation: function(msg) {
		const num = msg.num;
		state.ownPosts[num] = num;
		this.model.set({num: num});
		this.flushPending();
		var header = $(common.flatten(main.oneeSama.atama(msg)).join(''));
		this.$meta.replaceWith(header);
		this.$meta = header;
		if (!this.isThread)
			this.$el.addClass('editing');

		/*
		 TODO: Hide threads that are over THREADS_PER_PAGE. Also would need to be
		 removed from syncs client and server-side. Hmm.
		 */

		this.$el.attr('id', num);

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
	insertUploaded: function(info) {
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
	dispatch: function(msg) {
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

	onImageAllocation: function(msg) {
		const attrs = this.model.attributes;
		if (attrs.cancelled)
			return;
		if (!attrs.num && !attrs.sentAllocRequest) {
			main.send([common.INSERT_POST, this.allocationMessage(null, msg)]);
			this.model.set({sentAllocRequest: true});
		}
		else {
			main.send([common.INSERT_IMAGE, msg]);
		}
	},

	uploadError: function(msg) {
		if (this.model.get('cancelled'))
			return;
		this.model.set({
			uploadStatus: msg,
			uploading: false
		});
		if (this.$uploadForm)
			this.$uploadForm.find('input[name=alloc]').remove();
	},

	uploadStatus: function(msg) {
		if (this.model.get('cancelled'))
			return;
		this.model.set('uploadStatus', msg);
	},

	addReference: function(num, sel) {
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

	remove: function() {
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
	autoExpandImage: function() {}
});

function spoilerPaneUrl(sp) {
	return main.config.MEDIA_URL + 'spoil/spoil' + sp + '.png';
}

// Preload the spoiler panes for smoother display
function preloadPanes() {
	main.config.SPOILER_IMAGES.forEach(function(spoiler) {
		new Image().src = spoilerPaneUrl(spoiler);
	});
}

function imageUploadURL() {
	return (main.config.UPLOAD_URL || '../upload/')
		+ '?id=' + state.page.get('connID');
}

main.openPostBox = function(num) {
	var $a = main.$threads.find('#' + num);
	postSM.feed('new',
		$a.is('section') ? $a.children('aside') : $a.siblings('aside'));
};

window.addEventListener('message', function(event) {
	const msg = event.data;
	if (msg !== 'OK' && postForm)
		postForm.uploadError(msg);
}, false);

//Adds a followLock check for finishing posts 
(function () {
	var CV = ComposerView.prototype;
	CV.finish_wrapped = _.wrap(CV.finish, scroll.followLock);
})();

//Drag and Drop Functionality
function dragonDrop(e) {
    e.stopPropagation();
    e.preventDefault();
    var files = e.dataTransfer.files;
    if (!files.length)
            return;
    if (!postForm) {
            scroll.followLock(function () {
                    if (thread)
                            main.openPostBox(thread);
                    else {
                            var $s = $(e.target).closest('section');
                            if (!$s.length)
                                    return;
                            main.openPostBox($s.attr('id'));
                    }
            });
    }
    else {
            var attrs = postForm.model.attributes;
            if (attrs.uploading || attrs.uploaded)
                    return;
    }

    if (files.length > 1) {
            postForm.uploadError('Too many files.');
            return;
    }
    
    // Drag and drop does not supply a fakepath to file, so we have to use
    // a separate upload form from the postForm one. Meh.
    var extra = postForm.prepareUpload();
    var fd = new FormData();
    fd.append('image', files[0]);
    for (var k in extra)
            fd.append(k, extra[k]);
    // Can't seem to jQuery this shit
    var xhr = new XMLHttpRequest();
    xhr.open('POST', imageUploadURL());
    xhr.setRequestHeader('Accept', 'application/json');
    xhr.onreadystatechange = upload_shita;
    xhr.send(fd);

    postForm.notifyUploading();
}

function upload_shita() {
        if (this.readyState != 4 || this.status == 202)
                return;
        var err = this.responseText;
        // Everything just fine. Don't need to report.
        if (/legitimate imager response/.test(err))
                return;
        postForm.uploadError(err);
}

function stop_drag(e) {
        e.stopPropagation();
        e.preventDefault();
}

function setupUploadDrop(e) {
        function go(nm, f) { e.addEventListener(nm, f, false); }
        go('dragenter', stop_drag);
        go('dragexit', stop_drag);
        go('dragover', stop_drag);
        go('drop', dragonDrop);
}

$(function () {
        setupUploadDrop(document.body);
});