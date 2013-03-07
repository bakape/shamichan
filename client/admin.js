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

var multiSelecting = false;

function toggle_multi_selecting($post) {
	var oldTarget;
	if ($post) {
		oldTarget = lockTarget;
		set_lock_target(extract_num($post));
	}
	with_dom(function () {
	multiSelecting = !multiSelecting;
	if (multiSelecting) {
		$('body').addClass('multi-select');
		make_selection_handle().prependTo('article');
		make_selection_handle().prependTo('section > header');
		if ($post)
			$post.find('.select-handle:first'
					).addClass('selected');
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

$(function () {
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

	oneeSama.hook('afterInsert', function (target) {
		if (multiSelecting)
			make_selection_handle().prependTo(target);
	});
	show_toolbox();
});

$DOC.on('click', 'a.mod.addr', function (event) {
	event.preventDefault();
	var $a = $(event.currentTarget);
	var existing = $a.data('view');
	if (existing) {
		existing.remove();
		$a.attr('href', '#').removeData('view');
		return;
	}
	var ip = $a.prop('title') || $a.text();
	if (!ip || ip == '<bad IP>')
		return;
	var addr = window.addrs.get(ip);
	if (!addr) {
		send([101, ip]);
		addr = new Address({ip: ip, shallow: true});
		window.addrs.add(addr);
	}
	var view = new AddressView({model: addr})
	$a.removeAttr('href').data('view', view).append(view.render().el);
});

var Address = Backbone.Model.extend({
	idAttribute: 'ip',
	defaults: {
		count: 1,
	},
});

var AddressView = Backbone.View.extend({
	className: 'mod address',
	initialize: function () {
		this.listenTo(this.model, 'change', this.render);
	},

	render: function () {
		var attrs = this.model.attributes;
		if (attrs.shallow) {
			this.$el.hide();
			return this;
		}
		this.$el.show();
		this.$el.text(this.model.get('ip'));
		return this;
	},
});

var Addresses = Backbone.Collection.extend({
	model: Address,
	comparator: function (a) { return ip_mnemonic(a.ip); },
});

window.addrs = new Addresses;

var $panel;

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
		this.listenTo(this.model, 'change:uptime', this.renderUptime);
		$('<div/>', {id: 'ips'}).appendTo(this.el);
		$('<div/>', {id: 'mem'}).appendTo(this.el);
		$('<div/>', {id: 'uptime'}).appendTo(this.el);
	},

	renderVis: function (model, vis) {
		this.$el.toggle(!!vis);
	},

	renderIPs: function () {
		var $ips = this.$('#ips').empty();
		window.addrs.forEach(function (addr) {
			var attrs = addr.attributes;
			if (!attrs.count)
				return;
			var text = ip_mnemonic(attrs.ip);
			if (attrs.count > 1)
				text += ' (' + attrs.count + ')';
			$('<a/>', {
				"class": 'mod addr',
				href: '#',
				text: text,
				title: attrs.ip,
			}).appendTo($ips);
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
