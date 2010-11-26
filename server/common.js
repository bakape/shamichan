var DEFINES = exports;
DEFINES.INVALID = 0;
DEFINES.ALLOCATE_POST = 1;
DEFINES.INSERT_POST = 2;
DEFINES.UPDATE_POST = 3;
DEFINES.FINISH_POST = 4;
DEFINES.SYNCHRONIZE = 5;
DEFINES.INSERT_IMAGE = 6;

DEFINES.ANON = 'Anonymous';
DEFINES.INPUT_MIN_SIZE = 10;
DEFINES.MAX_POST_LINES = 30;
DEFINES.MAX_POST_CHARS = 2000;

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

function safe(frag) {
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
		/* anchor refs */
		var morcels = bits[i].split(/>>(\d+)/);
		for (var j = 0; j < morcels.length; j++) {
			if (j % 2)
				env.format_link(parseInt(morcels[j]), env);
			else if (i % 2)
				env.callback(morcels[j] + ' ');
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
			else if (state[0] === 0 && line[0] == '>')
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
	var m = text.match(/^(.{40}).{8,}(\.\w{3,4})$/);
	if (!m)
		return text;
	return [safe('<abbr title="'), text, safe('">'), m[1],
		safe('(&hellip;)'), m[2], safe('</abbr>')];
}

function gen_image(info, dirs) {
	var src = dirs.src_url + info.src;
	return [safe('<figure data-MD5="' + info.MD5 + '">' +
		'<figcaption>Image <a href="' + src + '" target="_blank">' +
		info.src + '</a> (' + info.size + ', ' + info.dims[0] +
		'x' + info.dims[1] + ', '), shorten_filename(info.imgnm),
		safe(')</figcaption><a href="' + src + '" target="_blank">' +
		'<img src="' + dirs.thumb_url + info.thumb + '" width="' +
		info.dims[2] + '" height="' + info.dims[3] + '"></a>' +
		'</figure>\n\t')];
}

function readable_time(time) {
	var d = new Date(time - new Date().getTimezoneOffset() * 60000);
	function pad(n) { return (n < 10 ? '0' : '') + n; }
	return (d.getUTCFullYear() + '/' + pad(d.getUTCMonth()+1) + '/' +
		pad(d.getUTCDate()) + ' ' + pad(d.getUTCHours()) + ':' +
		pad(d.getUTCMinutes()));
}
exports.readable_time = readable_time;

function datetime(time) {
	var d = new Date(time);
	function pad(n) { return (n < 10 ? '0' : '') + n; }
	return (d.getUTCFullYear() + '-' + pad(d.getUTCMonth()+1) + '-' +
		pad(d.getUTCDate()) + 'T' + pad(d.getUTCHours()) + ':' +
		pad(d.getUTCMinutes()) + ':' + pad(d.getUTCSeconds()) + 'Z');
}

function post_url(post, quote) {
	return (post.op || post.num) + (quote ? '#q' : '#') + post.num;
}
exports.post_url = post_url;

function num_html(post) {
	return ('<a href="' + post_url(post, false) + '">No.</a><a href="'
			+ post_url(post, true) + '">' + post.num + '</a>');
}

function gen_post(data, env) {
	var header = [safe('<header><b>'), data.name || DEFINES.ANON];
	if (data.trip)
		header.push(safe('</b> <code>' + data.trip + '</code>'));
	else
		header.push(safe('</b>'));
	if (data.email) {
		header.unshift(safe('<a class="email" href="mailto:'
				+ escape(data.email) + '">'));
		header.push(safe('</a>'));
	}
	header.push(safe(' <time pubdate datetime="' + datetime(data.time) +
			'">' + readable_time(data.time) + '</time> ' +
			num_html(data) + '</header>\n\t'));
	var body = [safe('<blockquote>'), format_body(data.body, env),
			safe('</blockquote>')];
	if (!data.image)
		return {header: header, body: body};
	var image = gen_image(env.image_view(data.image, data.imgnm, data.op),
			env.dirs);
	return {header: header, image: image, body: body};
}

exports.gen_post_html = function (data, env) {
	var o = safe(data.editing
			? '\t<article id="' + data.num + '" class="editing">'
			: '\t<article id="' + data.num + '">'),
	    c = safe('</article>\n'),
	    gen = gen_post(data, env);
	return flatten([o, gen.header, gen.image || '', gen.body, c]).join('');
};

exports.gen_thread = function (data, env) {
	var o = safe('<section id="' + data.num + '">'),
	    c = safe('</section>\n'),
	    gen = gen_post(data, env);
	return flatten([o, gen.image || '', gen.header, gen.body, '\n', c]);
};

exports.parse_name = function (name) {
	var tripcode = '', secure = '';
	var hash = name.indexOf('#');
	if (hash >= 0) {
		tripcode = name.substr(hash+1);
		name = name.substr(0, hash);
		hash = tripcode.indexOf('#');
		if (hash >= 0) {
			secure = tripcode.substr(hash+1);
			tripcode = tripcode.substr(0, hash);
		}
	}
	return [name.trim().substr(0, 100), tripcode.substr(0, 128),
			secure.substr(0, 128)];
};
