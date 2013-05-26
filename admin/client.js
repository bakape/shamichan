/* NOTE: This file is processed by server/state.js
         and served by server/server.js (to auth'd users only) */

yepnope(mediaURL + 'css/mod-v1.css');

var $selectButton, $controls;
window.loggedInUser = IDENT.email;
window.x_csrf = IDENT.csrf;

function show_toolbox() {
	var specs = [
		{name: 'Lewd', kind: 7},
		{name: 'Porn', kind: 8},
		{name: 'Delete', kind: 9},
		{name: 'Lock', kind: 11},
	];
	if (IDENT.auth == 'Admin')
		specs.push({name: 'Panel', kind: 'panel'});
	var $toolbox = $('<div/>', {id: 'toolbox', "class": 'mod'});

	$selectButton = $('<input />', {
		type: 'button', val: 'Select',
		click: function (e) { toggle_multi_selecting(); },
	});
	$toolbox.append($selectButton, ' ');

	$controls = $('<span></span>').hide();
	_.each(specs, function (spec) {
		$controls.append($('<input />', {
			type: 'button',
			val: spec.name,
			data: {kind: spec.kind},
		}), ' ');
	});
	$controls.on('click', 'input[type=button]', tool_action);

	_.each(delayNames, function (when, i) {
		var id = 'delay-' + when;
		var $label = $('<label/>', {text: when, 'for': id});
		var $radio = $('<input/>', {
			id: id, type: 'radio', val: when, name: 'delay',
		});
		if (i == 0)
			$radio.prop('checked', true);
		$controls.append($radio, $label, ' ');
	});

	$toolbox.append($controls).insertBefore(THREAD ? 'hr:last' : $ceiling);
}

function tool_action(event) {
	var ids = [];
	var $sel = $('.selected');
	$sel.each(function () {
		var id = extract_num(parent_post($(this)));
		if (id)
			ids.push(id);
	});
	var $button = $(this);
	var kind = $button.data('kind');
	if (kind == 'panel')
		return toggle_panel();

	/* On a thread page there's only one thread to lock, so... */
	if (kind == 11 && THREAD && !ids.length)
		ids = [THREAD];

	if (ids.length) {
		var when = $('input:radio[name=delay]:checked').val();
		ids.unshift(parseInt(kind, 10), {when: when});
		send(ids);
		$sel.removeClass('selected');
	}
	else {
		var orig = $button.val();
		var caption = _.bind($button.val, $button);
		caption('Nope.');
		if (orig != 'Nope.')
			_.delay(caption, 2000, orig);
	}
}

readOnly.push('graveyard');
menuOptions.unshift('Select');
menuHandlers.Hide = function () { alert('nope.avi'); };

var multiSelecting = false;

function toggle_multi_selecting(num, $post) {
	var oldTarget;
	if ($post) {
		oldTarget = lockTarget;
		set_lock_target(num);
	}
	with_dom(function () {
	multiSelecting = !multiSelecting;
	if (multiSelecting) {
		$('body').addClass('multi-select');
		make_selection_handle().prependTo('article');
		make_selection_handle().prependTo('section > header');
		if ($post)
			select_post($post);
		$controls.show();
		$selectButton.val('X');
	}
	else {
		$('body').removeClass('multi-select');
		$('.select-handle').remove();
		$controls.hide();
		$selectButton.val('Select');
	}
	});
	if ($post)
		set_lock_target(oldTarget);
}

menuHandlers.Select = toggle_multi_selecting;

function enable_multi_selecting() {
	if (!multiSelecting)
		toggle_multi_selecting();
}

function select_post($post) {
	$post.find('.select-handle:first').addClass('selected');
}

function make_selection_handle() {
	return $('<a class="select-handle" href="#"/>');
}

window.fun = function () {
	send([33, THREAD]);
};

override(ComposerView.prototype, 'make_alloc_request',
			function (orig, text, img) {
	var msg = orig.call(this, text, img);
	if ($('#authname').is(':checked'))
		msg.auth = IDENT.auth;
	return msg;
});

$DOC.on('click', '.select-handle', function (event) {
	event.preventDefault();
	$(event.target).toggleClass('selected');
});

with_dom(function () {
	$('h1').text('Moderation - ' + $('h1').text());
	var $authname = $('<input>', {type: 'checkbox', id: 'authname'});
	var $label = $('<label/>', {text: ' '+IDENT.auth}).prepend($authname);
	$name.after(' ', $label);

	/* really ought to be done with model observation! */
	$authname.change(function () {
		if (postForm)
			postForm.propagate_ident();
	});
	oneeSama.hook('fillMyName', function ($el) {
		var auth = $('#authname').is(':checked');
		$el.toggleClass(IDENT.auth.toLowerCase(), auth);
		if (auth)
			$el.append(' ## ' + IDENT.auth)
	});

	Backbone.on('afterInsert', function ($el) {
		if (multiSelecting)
			make_selection_handle().prependTo($el);
	});
	show_toolbox();
});

var Address = Backbone.Model.extend({
	idAttribute: 'ip',
	defaults: {
		count: 0,
	},
});

var AddressView = Backbone.View.extend({
	className: 'mod address',
	events: {
		'keydown .name': 'entered_name',
		'click .sel-all': 'select_all',
	},

	initialize: function () {
		var $el = this.$el;
		$('<span/>', {"class": 'ip'}).appendTo($el);
		$el.append(' &nbsp; ', $('<input/>', {
			"class": 'sel-all',
			type: 'button',
			val: 'Sel All'
		}));
		$el.append(
			'<br>',
			$('<input>', {"class": 'name', placeholder: 'Name'})
		);
		this.listenTo(this.model, 'change', this.render);
	},

	render: function () {
		var attrs = this.model.attributes;
		if (attrs.shallow) {
			this.$('.ip').text('Loading...');
			return this;
		}
		this.$('.ip').text(attrs.ip);
		var $name = this.$('.name');
		if (!this.focusedName) {
			_.defer(function () {
				$name.focus().prop({
					selectionStart: 0,
					selectionEnd: $name.val().length,
				});
			});
			this.focusedName = true;
		}
		if (attrs.name != $name.val()) {
			$name.val(attrs.name);
		}
		return this;
	},

	entered_name: function (event) {
		if (event.which != 13)
			return;
		event.preventDefault();
		var name = this.$('.name').val().trim();
		send([102, this.model.id, name]);
		this.remove();
	},

	remove: function () {
		this.trigger('preremove');
		Backbone.View.prototype.remove.call(this);
	},

	select_all: function () {
		if (!THREAD)
			return alert('TODO');
		var models = Threads.get(THREAD).get('replies').where({
			ip: this.model.get('ip'),
		});
		if (!models.length)
			return;
		enable_multi_selecting();
		with_dom(function () {
			$.each(models, function () {
				select_post($('#' + this.id));
			});
		});
		this.remove();
	},
});

// basically just a link
var AddrView = Backbone.View.extend({
	tagName: 'a',
	className: 'mod addr',

	events: {
		click: 'toggle_expansion',
	},

	initialize: function () {
		this.$el.attr('href', '#');
		this.listenTo(this.model, 'change:name', this.render);
	},

	render: function () {
		var attrs = this.model.attributes;
		var text = ip_mnemonic(attrs.ip);
		if (attrs.name)
			text += ' "' + attrs.name + '"';
		this.$el.attr('title', attrs.ip).text(text);
		return this;
	},

	toggle_expansion: function (event) {
		if (event.target !== this.el)
			return;
		event.preventDefault();

		if (this.expansion)
			return this.expansion.remove();

		this.expansion = new AddressView({model: this.model});
		this.$el.after(this.expansion.render().el);
		this.listenTo(this.expansion, 'preremove',
				this.expansion_removed);

		if (this.model.get('shallow'))
			send([101, this.model.id]);
	},

	remove: function () {
		if (this.expansion)
			this.expansion.remove();
		Backbone.View.prototype.remove.call(this);
	},

	expansion_removed: function () {
		this.expansion = null;
	},
});

var Addresses = Backbone.Collection.extend({
	model: Address,
	comparator: function (a) { return ip_mnemonic(a.ip); },
});

window.addrs = new Addresses;

function hook_up_address($post, op) {
	var $a = $post.find('a.mod.addr:first');
	if (!$a.length)
		return;
	var ip = $a.prop('title') || $a.text();
	var givenName;
	var m = $a.text().match(/^([\w'.]+) "(.+)"$/);
	if (m) {
		if (is_valid_ip(m[1]))
			ip = m[1];
		givenName = m[2];
	}
	if (!is_valid_ip(ip))
		return;

	/* Activate this address link */
	var address = window.addrs.get(ip);
	if (!address) {
		address = new Address({ip: ip});
		address.set(givenName ? {name: givenName} : {shallow: true});
		window.addrs.add(address);
	}
	var view = new AddrView({model: address, el: $a[0]});
	if (address.get('name'))
		view.render();

	/* Augment post with IP */
	var thread = Threads.get(op);
	if (thread) {
		var num = extract_num($post);
		var post = num==op ? thread : thread.get('replies').get(num);
		if (post && !post.has('ip'))
			post.set('ip', ip);
	}
}
Backbone.on('afterInsert', hook_up_address);

with_dom(function () {
	$('section').each(function () {
		var $section = $(this);
		var op = extract_num($section);
		hook_up_address($section, op);
		$section.find('article').each(function () {
			hook_up_address($(this), op);
		});
	});
});

window.adminState = new Backbone.Model({
});

var PanelView = Backbone.View.extend({
	id: 'panel',

	initialize: function () {
		this.listenTo(this.model, 'change:visible', this.renderVis);
		this.listenTo(window.addrs, 'add change:count reset',
				this.renderIPs);
		this.listenTo(this.model, 'change:memoryUsage',
				this.renderMemory);
		this.listenTo(this.model, 'change:addrs change:bans',
				this.renderCounts);
		this.listenTo(this.model, 'change:uptime', this.renderUptime);
		$('<div/>', {id: 'ips'}).appendTo(this.el);
		$('<div/>', {id: 'mem'}).appendTo(this.el);
		$('<div/>', {id: 'counts'}).appendTo(this.el);
		$('<div/>', {id: 'uptime'}).appendTo(this.el);
	},

	renderVis: function (model, vis) {
		this.$el.toggle(!!vis);
	},

	renderIPs: function () {
		var $ips = this.$('#ips').empty();
		window.addrs.forEach(function (addr) {
			var n = addr.get('count');
			if (!n)
				return;
			var el = new AddrView({model: addr}).render().el;
			$ips.append(el, n>1 ? ' ('+n+')' : '', '<br>');
		});
	},

	renderMemory: function (model, mem) {
		function mb(n) {
			return Math.round(n/1000000);
		}
		this.$('#mem').text(
			mb(mem.heapUsed) + '/' + mb(mem.heapTotal) +
			' MB heap used.'
		);
	},

	renderCounts: function (model) {
		var a = model.attributes;
		this.$('#counts').text(pluralize(a.addrs, 'addr') + ', ' +
				pluralize(a.bans, 'ban') + '.');
	},

	renderUptime: function (model, s) {
		var m = Math.floor(s / 60) % 60;
		var h = Math.floor(s / 3600) % 60;
		var d = Math.floor(s / (3600*24));
		h = h ? h+'h' : '';
		d = d ? d+'d' : '';
		this.$('#uptime').text('Up '+ d + h + m +'m.');
	},
});

function toggle_panel() {
	var show = !adminState.get('visible');
	send([show ? 60 : 61, 'adminState']);
}

if (IDENT.auth == 'Admin') (function () {
	var $panel = $('<div/>', {id: 'panel', "class": 'mod'}).hide();
	var view = new PanelView({model: adminState, el: $panel[0]});
	$panel.appendTo('body');
})();
