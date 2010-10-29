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

function break_long_words(frag, env) {
	if (frag.safe) {
		env.callback(frag);
		return;
	}
	var bits = frag.split(/(\S{60})/);
	for (var i = 0; i < bits.length; i++) {
		if (i % 2) {
			env.callback(safe('&shy;'));
			continue;
		}
		/* anchor refs */
		var morcels = bits[i].split(/>>(\d+)/);
		for (var j = 0; j < morcels.length; j++) {
			if (j % 2)
				env.format_link(parseInt(morcels[j]), env);
			else
				env.callback(morcels[j]);
		}
	}
}

function format_fragment(frag, state, env) {
	function do_transition(token, new_state) {
		if (state[0] == 1 && new_state != 1)
			env.callback(safe('</em>'));
		switch (new_state) {
		case 1:
			if (state[0] != 1) {
				env.callback(safe('<em>'));
				state[0] = 1;
			}
			break_long_words(token, env);
			break;
		case 3:
			if (token[1] == '/') {
				state[1]--;
				env.callback(safe('</del>'));
			}
			else {
				env.callback(safe('<del>'));
				state[1]++;
			}
			break;
		default:
			break_long_words(token, env);
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

function format_body(body, env) {
	var state = initial_post_state();
	var output = [];
	env.callback = function (frag) { output.push(frag); }
	format_fragment(body, state, env);
	env.callback = null;
	if (state[0] == 1)
		output.push(safe('</em>'));
	for (var i = 0; i < state[1]; i++)
		output.push(safe('</del>'));
	return output;
}

function shorten_filename(text) {
	m = text.match(/^(.{20}).{8,}(\.\w{3,4})$/);
	return m ? [m[1], safe('[&hellip;]'), m[2]] : text;
}

function image_metadata(info) {
	var srcNm = info.src.substr(info.src.lastIndexOf('/') + 1);
	return [safe('<span>Image <a href="'+info.src+'" target="_blank">'),
		srcNm, safe('</a> (' + info.size + ', ' + info.dims[0] + 'x'
		+ info.dims[1] + ', <abbr title="'), info.name, safe('">'),
		shorten_filename(info.name), safe('</abbr>)</span>')];
}

function thumbnail_html(info) {
	return '<a href="' + info.src + '" target="_blank"><img src="'
		+ info.thumb + '" width="' + info.dims[2] + '" height="'
		+ info.dims[3] + '" data-MD5="' + info.MD5 + '"></a>';
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

function time_tag_html(time) {
	return ('<time pubdate datetime="' + datetime(time) + '">'
		+ readable_time(time) + '</time>');
}

function post_url(post) {
	return (post.op || post.num) + '#q' + post.num;
}
exports.post_url = post_url;

exports.gen_post_html = function (data, env) {
	var edit = data.editing ? '" class="editing' : '';
	var ident = [safe('<b>'), data.name, safe('</b>')];
	if (data.trip) {
		ident[2].safe += ' <code>';
		ident.push(data.trip);
		ident.push(safe('</code>'));
	}
	if (data.email) {
		ident.unshift(safe('<a class="email" href="mailto:'
				+ escape(data.email) + '">'));
		ident.push(safe('</a>'));
	}
	var image = data.image ? [image_metadata(data.image), safe('</header>'),
			safe(thumbnail_html(data.image))] : safe('</header>');
	var post = [safe('\t<article id="q' + data.num + edit + '"><header>'),
		ident, safe(' ' + time_tag_html(data.time) + ' '),
		safe('<a href="#q' + data.num + '">No.</a><a href="'
			+ post_url(data) + '">' + data.num + '</a>'),
		image, safe('\n\t\t<blockquote>'), format_body(data.body, env),
		safe('</blockquote></article>\n')];
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
