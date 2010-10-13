exports.INVALID = 0;
exports.ALLOCATE_POST = 1;
exports.INSERT_POST = 2;
exports.UPDATE_POST = 3;
exports.FINISH_POST = 4;
exports.SYNCHRONIZE = 5;

function escape_html(html) {
	return html.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(
		/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escape_fragment(frag) {
	var t = typeof(frag);
	if (t == 'object' && typeof(frag.safe) == 'string')
		return frag.safe;
	else if (t == 'string')
		return escape_html(frag);
	else if (t == 'number')
		return frag.toString();
	else
		return '???';

}
exports.escape_fragment = escape_fragment;

function flatten(frags) {
	var out = [];
	for (var i = 0; i < frags.length; i++) {
		var frag = frags[i];
		if (frag.constructor == Array)
			out = out.concat(flatten(frag));
		else
			out.push(escape_fragment(frag));
	}
	return out;
}

safe = function (frag) {
	return {safe: frag};
}
exports.safe = safe;

function map_unsafe(frags, func) {
	for (var i = 0; i < frags.length; i++) {
		if (typeof(frags[i]) == 'string')
			frags[i] = func(frags[i]);
		else if (frags[i].constructor == Array)
			frags[i] = map_unsafe(frags[i], func);
	}
	return frags;
}

function initial_post_state() {
	return [0, 0];
}
exports.initial_post_state = initial_post_state;

function format_fragment(frag, state, func) {
	if (!func)
		func = function (tok) {};
	function do_transition(token, new_state) {
		if (state[0] == 1 && new_state != 1)
			func(safe('</em>'));
		switch (new_state) {
		case 1:
			if (state[0] != 1) {
				func(safe('<em>'));
				state[0] = 1;
			}
			func(token);
			break;
		case 3:
			if (token[1] == '/') {
				state[1]--;
				func(safe('</del>'));
			}
			else {
				func(safe('<del>'));
				state[1]++;
			}
			break;
		default:
			func(token);
			break;
		}
		state[0] = new_state;
	}
	var chunks = frag.split(/(\[\/?spoiler\])/i);
	for (var i = 0; i < chunks.length; i++) {
		var chunk = chunks[i];
		if (i % 2) {
			var new_state = 3;
			if (chunk[1] == '/' && state[1] < 1)
				new_state = (state[0] == 1) ? 1 : 2;
			do_transition(chunk, new_state);
			continue;
		}
		lines = chunk.split(/(\n)/);
		for (var l = 0; l < lines.length; l++) {
			var line = lines[l];
			if (l % 2)
				do_transition(safe('<br>'), 0);
			else if (state[0] == 0 && line[0] == '>')
				do_transition(line, 1);
			else if (line)
				do_transition(line, (state[0] == 1) ? 1 : 2);
		}
	}
}
exports.format_fragment = format_fragment;

function format_body(body) {
	var state = initial_post_state();
	var output = [];
	format_fragment(body, state, function (frag) {
		output.push(frag);
	});
	if (state[0] == 1)
		output.push(safe('</em>'));
	for (var i = 0; i < state[1]; i++)
		output.push(safe('</del>'));
	return output;
}

function readable_time(time) {
	var d = new Date(time - new Date().getTimezoneOffset() * 60000);
	function pad(n) { return (n < 10 ? '0' : '') + n; }
	return (d.getUTCFullYear() + '/' + pad(d.getUTCMonth()+1) + '/' +
		pad(d.getUTCDate()) + ' ' + pad(d.getUTCHours()) + ':' +
		pad(d.getUTCMinutes()));
}

function datetime(time) {
	var d = new Date(time);
	function pad(n) { return (n < 10 ? '0' : '') + n; }
	function pad3(n) { return (n < 10 ? '00' : (n < 100 ? '0' : '')) + n; }
	return (d.getUTCFullYear() + '-' + pad(d.getUTCMonth()+1) + '-' +
		pad(d.getUTCDate()) + 'T' + pad(d.getUTCHours()) + ':' +
		pad(d.getUTCMinutes()) + ':' + pad(d.getUTCSeconds()) + 'Z');
}

exports.gen_post_html = function (data) {
	var edit = data.editing ? '" class="editing' : '';
	var time_tag = ' <time datetime="' + datetime(data.time) + '">';
	var email = safe('<b>'), email_end = safe(time_tag);
	if (data.email) {
		email = safe('<a class="email" href="mailto:'
				+ escape(data.email) + '"><b>');
		email_end = safe('</a>' + time_tag);
	}
	var trip = data.trip ? safe('</b> <code>' + data.trip + '</code>')
			: safe('</b>');
	var post = [safe('\t<li id="q' + data.num + edit + '"><span>'),
		email, data.name, trip, email_end, readable_time(data.time),
		safe('</time> No.<a href="#q' + data.num + '">' + data.num
			+ '</a></span> <blockquote>'),
		format_body(data.body), safe('</blockquote></li>\n')];
	return flatten(post).join('');
}

exports.parse_name = function (name) {
	var tripcode = '', secure = '';
	var hash = name.indexOf('#');
	if (hash >= 0 && hash < 128) {
		tripcode = name.substr(hash+1);
		name = name.substr(0, hash);
		hash = tripcode.indexOf('#');
		if (hash >= 0 && hash < 128) {
			secure = tripcode.substr(hash+1);
			tripcode = tripcode.substr(0, hash);
		}
	}
	return [name.trim() || 'Anonymous', tripcode, secure];
}

function clone (obj) {
	if (obj == null || typeof(obj) != 'object')
		return obj;
	var temp = new obj.constructor();
	for (var key in obj)
		temp[key] = clone(obj[key]);
	return temp;
}
exports.clone = clone;
