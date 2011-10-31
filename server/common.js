var DEFINES = exports;
DEFINES.INVALID = 0;
DEFINES.ALLOCATE_POST = 1;
DEFINES.INSERT_POST = 2;
DEFINES.UPDATE_POST = 3;
DEFINES.FINISH_POST = 4;
DEFINES.DELETE_POSTS = 5;
DEFINES.DELETE_THREAD = 6;
DEFINES.INSERT_IMAGE = 7;
DEFINES.IMAGE_STATUS = 8;
DEFINES.SYNCHRONIZE = 9;
DEFINES.CATCH_UP = 10;

DEFINES.ANON = 'Anonymous';
DEFINES.INPUT_MIN_SIZE = 300;
DEFINES.INPUT_ROOM = 20;
DEFINES.MAX_POST_LINES = 30;
DEFINES.MAX_POST_CHARS = 2000;
DEFINES.WORD_LENGTH_LIMIT = 120;

function is_pubsub(t) {
	return t >= INSERT_POST && t <= DELETE_THREAD;
}

var entities = {'&' : '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;'};
function escape_html(html) {
	return html.replace(/[&<>"]/g, function (c) {
		return entities[c];
	});
}
exports.escape_html = escape_html;

function escape_fragment(frag) {
	var t = typeof(frag);
	if (t == 'object' && frag && typeof(frag.safe) == 'string')
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

function is_empty(o) {
	if (!o)
		return true;
	for (k in o)
		if (o.hasOwnProperty(k))
			return false;
	return true;
}
exports.is_empty = is_empty;

function is_noko(email) {
	return email && email.indexOf('@') == -1 && email.match(/noko/i);
}
exports.is_noko = is_noko;
function is_sage(email) {
	return email && email.indexOf('@') == -1 && email.match(/sage/i);
}
exports.is_sage = is_sage;

var OneeSama = function (t) {
	this.tamashii = t;
};
exports.OneeSama = OneeSama;
var OS = OneeSama.prototype;

var break_re = new RegExp("(\\S{" + DEFINES.WORD_LENGTH_LIMIT + "})");
/* internal refs and youtube videos */
var ref_re = />>(\d+|>\/?(?:watch\?)?v[=\/][\w-]{11}(?:#t=[\dhms]{1,9})?)/;
var youtube_re = /^>>>\/?(?:watch\?)?v[=\/]([\w-]{11})(#t=[\dhms]{1,9})?$/;
var youtube_time_re = /^#t=(?:(\d\d?)h)?(?:(\d\d?)m)?(?:(\d\d?)s)?$/;
var youtube_url_re = /(?:>>>*?)?(?:http:\/\/)?(?:www\.)?youtube\.com\/watch\?((?:[^\s#&=]+=[^\s#&]*&)*)?v=([\w-]{11})((?:&[^\s#&=]+=[^\s#&]*)*)&?(#t=[\dhms]{1,9})?/;

OS.break_heart = function (frag) {
	if (frag.safe)
		return this.callback(frag);
	var bits = frag.split(break_re);
	for (var i = 0; i < bits.length; i++) {
		/* anchor refs */
		var morsels = bits[i].split(ref_re);
		for (var j = 0; j < morsels.length; j++) {
			var m = morsels[j];
			if (j % 2) {
				if (m[0] == '>') {
					/* This is alright since it's always
					   a single word */
					this.callback(safe('<cite>' +
							escape_html('>>' + m) +
							'</cite>'));
				}
				else
					this.tamashii(parseInt(m, 10));
			}
			else if (i % 2)
				this.callback(m + ' ');
			else
				this.callback(m);
		}
	}
};

OS.iku = function (token, to) {
	var state = this.state;
	if (state[0] == 1 && to != 1)
		this.callback(safe('</em>'));
	switch (to) {
	case 1:
		if (state[0] != 1) {
			this.callback(safe('<em>'));
			state[0] = 1;
		}
		this.break_heart(token);
		break;
	case 3:
		if (token[1] == '/') {
			state[1]--;
			this.callback(safe('</del>'));
		}
		else {
			this.callback(safe('<del>'));
			state[1]++;
		}
		break;
	default:
		this.break_heart(token);
		break;
	}
	state[0] = to;
}

OS.fragment = function (frag) {
	var chunks = frag.split(/(\[\/?spoiler\])/i);
	var state = this.state;
	for (var i = 0; i < chunks.length; i++) {
		var chunk = chunks[i];
		if (i % 2) {
			var to = 3;
			if (chunk[1] == '/' && state[1] < 1)
				to = (state[0] == 1) ? 1 : 2;
			this.iku(chunk, to);
			continue;
		}
		lines = chunk.split(/(\n)/);
		for (var l = 0; l < lines.length; l++) {
			var line = lines[l];
			if (l % 2)
				this.iku(safe('<br>'), 0);
			else if (state[0] === 0 && line[0] == '>')
				this.iku(line, 1);
			else if (line)
				this.iku(line, state[0]==1 ? 1 : 2);
		}
	}
};

OS.karada = function (body) {
	var output = [];
	this.state = [0, 0];
	this.callback = function (frag) { output.push(frag); }
	this.fragment(body);
	this.callback = null;
	if (this.state[0] == 1)
		output.push(safe('</em>'));
	for (var i = 0; i < this.state[1]; i++)
		output.push(safe('</del>'));
	return output;
}

function chibi(text) {
	var m = text.match(/^(.{40}).{8,}(\.\w{3,4})$/);
	/* Comma inlined for convience in OS.gazou (beware of concatenating
	 * lists with strings) */
	if (!m)
		return ', ' + text;
	return [safe(', <abbr title="'), text, safe('">'), m[1],
		safe('(&hellip;)'), m[2], safe('</abbr>')];
}

OS.gazou = function (info) {
	var src = encodeURI(this.media + 'src/' + info.src),
		thumb = encodeURI(this.media + 'thumb/' + info.thumb),
		d = info.dims;
	return [safe('<figure data-MD5="' + info.MD5 + '">' +
		'<figcaption>Image <a href="' + src + '" target="_blank">' +
		info.src + '</a> (' + readable_filesize(info.size) + ', ' +
		d[0] + 'x' + d[1]), this.full ? chibi(info.imgnm) : '',
		safe(')</figcaption><a href="' + src + '" target="_blank">' +
		'<img src="' + thumb + '" width="' +
		d[2] + '" height="' + d[3] + '"></a>' + '</figure>\n\t')];
};

function readable_filesize(size) {
       /* Metric. Deal with it. */
       if (size < 1000)
               return size + ' B';
       if (size < 1000000)
               return Math.round(size / 1000) + ' KB';
       size = Math.round(size / 100000).toString();
       return size.slice(0, -1) + '.' + size.slice(-1) + ' MB';
}

function pad(n) {
	return (n < 10 ? '0' : '') + n;
}

function readable_time(time) {
	var d = new Date(time - new Date().getTimezoneOffset() * 60000);
	return (d.getUTCFullYear() + '/' + pad(d.getUTCMonth()+1) + '/' +
		pad(d.getUTCDate()) + ' ' + pad(d.getUTCHours()) + ':' +
		pad(d.getUTCMinutes()));
}
exports.readable_time = readable_time;

function datetime(time) {
	var d = new Date(time);
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

function expand_html(num) {
	return ' &nbsp; [<a href="' + num + '" class="expand">Expand</a>]';
}

OS.monogatari = function (data) {
	var auth = data.auth;
	var header = auth ? [safe('<b class="'),auth.toLowerCase(),safe('">')]
			: [safe('<b>')];
	header.push(data.name || DEFINES.ANON);
	if (data.trip)
		header.push(safe(' <code>' + data.trip + '</code>'));
	if (auth)
		header.push(' ## ' + auth);
	header.push(safe('</b>'));
	if (data.email) {
		header.unshift(safe('<a class="email" href="mailto:'
				+ encodeURI(data.email) + '">'));
		header.push(safe('</a>'));
	}
	header.unshift(safe('<header>'));
	header.push(safe(' <time pubdate datetime="' + datetime(data.time) +
			'">' + readable_time(data.time) + '</time> ' +
			num_html(data)));
	if (!this.full && !data.op)
		header.push(safe(expand_html(data.num)));
	header.push(safe('</header>\n\t'));
	var body = [safe('<blockquote>'), this.karada(data.body),
			safe('</blockquote>')];
	if (!data.image)
		return {header: header, body: body};
	return {header: header, image: this.gazou(data.image), body: body};
};

OS.mono = function (data) {
	var o = safe(data.editing
			? '\t<article id="' + data.num + '" class="editing">'
			: '\t<article id="' + data.num + '">'),
	    c = safe('</article>\n'),
	    gen = this.monogatari(data);
	return flatten([o, gen.header, gen.image || '', gen.body, c]).join('');
};

OS.monomono = function (data) {
	var o = safe('<section id="' + data.num +
		(data.hctr ? '" data-sync="'+data.hctr : '') +
		(data.full ? '' : '" data-imgs="'+data.imgctr) + '">'),
	    c = safe('</section>\n'),
	    gen = this.monogatari(data);
	return flatten([o, gen.image || '', gen.header, gen.body, '\n', c]);
};

exports.abbrev_msg = function (omit, img_omit) {
	return omit + (omit==1 ? ' reply' : ' replies') + (img_omit
		? ' and ' + img_omit + ' image' + (img_omit==1 ? '' : 's')
		: '') + ' omitted.';
};

exports.parse_name = function (name) {
	var tripcode = '', secure = '';
	var hash = name.indexOf('#');
	if (hash >= 0) {
		tripcode = name.substr(hash+1);
		name = name.substr(0, hash);
		hash = tripcode.indexOf('#');
		if (hash >= 0) {
			secure = escape_html(tripcode.substr(hash+1));
			tripcode = tripcode.substr(0, hash);
		}
		tripcode = escape_html(tripcode);
	}
	return [name.trim().substr(0, 100), tripcode.substr(0, 128),
			secure.substr(0, 128)];
};
