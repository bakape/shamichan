/*
 * Evertything related to writing and commiting posts
 */

var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	client = require('../client'),
	common = require('../../common'),
	ident = require('./identity'),
	imager = require('./imager'),
	index = require('./index'),
	main = require('../main'),
	options = require('../options'),
	state = require('../state');

var connSM = main.connSM,
	postSM = main.postSM,
	uploadModel = new Backbone.Model(),
	postModel;

var postForm = exports.postForm,
// Minimal size of the input buffer
	inputMinSize = 300;

// For mobile
if (window.screen && screen.width <= 320)
	inputMinSize = 50;

var ComposerModel = index.PostModel.extend({
	status: 'inactive',
	mine: true
});

// Synchronyse postform state with websocket connectivity
connSM.on('synced', postSM.feeder('sync'));
connSM.on('dropped', postSM.feeder('desync'));
connSM.on('desynced', postSM.feeder('desync'));

postSM.act('* + desync -> none', function () {
	// TODO: Desync logic
});

postSM.act('none + sync, draft, alloc + done -> ready', function () {
	// TODO: Add unfinished post checking
});

// Make new postform
postSM.act('ready + new -> draft', function($aside) {
	var op = null;
	var $sec = $aside.closest('section');
	if ($sec.length)
		op = extractNum($sec);
	else
		$sec = $('<section/>');

	postModel = new ComposerModel({op: op});
	exports.postForm = new ComposerView({
		model: postModel,
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
	postForm.on_allocation(msg);
});

main.$doc.on('click', 'aside a', function() {
	postSM.feed('new', $(this).parent());
});

main.$doc.on('keydown', handle_shortcut);

function handle_shortcut(event) {
	if (!event.altKey)
		return;

	var used = false,
		opts = options.attributes;
	switch(event.which) {
		case opts.new:
			var $aside = state.page.get('thread') ? $('aside') : $ceiling().next();
			if ($aside.is('aside') && $aside.length == 1) {
				postSM.feed('new', $aside);
				used = true;
			}
			break;
		case opts.togglespoiler:
			if (postForm) {
				postForm.on_toggle(event);
				used = true;
			}
			break;
		case opts.done:
			if (postForm) {
				if (!postForm.submit.attr('disabled')) {
					postForm.finish_wrapped();
					used = true;
				}
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
			imager.massExpander.set('expand', !massExpander.get('expand'));
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
	return main.$threads.children('hr:first');
}

var ComposerView = Backbone.View.extend({
	initialize: function(args) {
		this.listenTo(this.model, 'change', this.renderButtons);
		this.listenTo(this.model, 'change:spoiler', this.renderSpoilerPane);

		this.setElement((this.model.get('op') ? $('<article/>') : args.$sec)[0]);

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
			else {
				this.callback(common.safe('<a class="nope">&gt;&gt;'
						+ num + '</a>'));
			}
		});
		// Initialise the renderer instance
		this.imouto.callback = client.inject;
		this.imouto.op = state.page.get('thread');
		this.imouto.state = [common.S_BOL, 0];
		// TODO: Convert current OneeSama.state array to more flexible object
		this.imouto.state2 = {spoiler: 0};
		this.imouto.buffer = this.$buffer;
		this.imouto.eLinkify = main.oneeSama.eLinkify;
		this.imouto.hook('spoilerTag', client.touchable_spoiler_tag);
		main.oneeSama.trigger('imouto', this.imouto);

		this.render(args);
		//this.$input.on('input', this.on_input.bind(this, undefined));
	},

	// Initial render
	render: function(args) {
		// A defined op means the post is a reply, not a new thread
		const op = !!this.model.get('op');

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
		if (!op) {
			this.$el.append('<label for="subject">Subject: </label>',
				this.$subject);
			this.$blockquote.hide();
		}
		this.$uploadForm = this.renderUploadForm();
		this.$el.append(this.$uploadForm);
		// Add a menu to the postform
		main.oneeSama.trigger('draft', this.$el);
		this.renderIdentity();
		args.$dest.replaceWith(this.$el);

		 if (op) {
			 this.resizeInput();
			 this.$input.focus();
		 }
		 else {
			 this.$el.after('<hr class="sectionHr"/>');
			 this.$subject.focus();
		 }
		 $('aside').remove();

		 preload_panes();
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
		var $tag = this.$meta.children('a:first');
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
			+ (sp ? spoiler_pane_url(sp)
				: main.imagerConfig.MEDIA_URL + 'css/ui/pane.png')
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
			accept: main.imagerConfig.WEBM ? 'imager/*;.webm' : 'image/*',
			change: $.proxy(this, 'on_image_chosen')
		});
		this.$toggle = $('<input/>', {
			type: 'button',
			id: 'toggle'
		});
		this.$uploadStatus = $('<strong/>');
		$form.append(this.$cancel, this.$imageInput, this.$toggle, ' ',
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

	resizeInput: function(val) {
		if (typeof val != 'string')
			val = this.$input.val();
		this.$sizer.text(val);
		var size = this.$sizer.width() + common.INPUT_ROOM;
		size = Math.max(size, inputMinSize
			- this.$input.offset().left - this.$el.offset().left);
		this.$input.css('width', size + 'px');
	}
});

function spoiler_pane_url(sp) {
	return main.imagerConfig.MEDIA_URL + 'spoil/spoil' + sp + '.png';
}

// Preload the spoiler panes for smoother display
function preload_panes() {
	main.imagerConfig.SPOILER_IMAGES.forEach(function(spoiler) {
		new Image().src = spoiler_pane_url(spoiler);
	});
}
