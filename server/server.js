var _ = require('./lib/underscore'),
    common = require('./common'),
    config = require('./config'),
    db = require('./db'),
    fs = require('fs'),
    http = require('http'),
    pix = require('./pix'),
    twitter = require('./twitter'),
    tripcode,
    url_parse = require('url').parse,
    util = require('util');

_.templateSettings = {
	interpolate: /\{\{(.+?)\}\}/g
};

var clients = {};
var dispatcher = {};
var indexTmpl, notFoundHtml, adminJs;

/* I always use encodeURI anyway */
escape = common.escape_html;

function Okyaku(socket) {
	this.ip = socket.handshake.address.address;
	this.id = socket.id;
	this.socket = socket;
	this.watching = {};
	this.db = new db.Yakusoku;
	this.skipped = 0;
	socket.on('message', this.on_message.bind(this));
	socket.on('disconnect', this.on_disconnect.bind(this));
	socket.on('error', console.error.bind(console, 'socket:'));
	this.db.on('error', console.error.bind(console, 'redis:'));
}
var OK = Okyaku.prototype;

OK.send = function (msg) {
	this.socket.send(JSON.stringify([msg]));
};

dispatcher[common.SYNCHRONIZE] = function (msg, client) {
	if (msg.length != 2)
		return false;
	var syncs = msg[0], live = msg[1];
	if (!syncs || typeof syncs != 'object')
		return false;
	if (client.synced) {
		console.error("warning: Client tried to sync twice");
		return false;
	}
	var dead_threads = [], count = 0;
	for (var k in syncs) {
		if (!k.match(/^\d+$/))
			return false;
		k = parseInt(k, 10);
		if (!k || typeof syncs[k] != 'number')
			return false;
		if (db.OPs[k] != k) {
			delete syncs[k];
			dead_threads.push(k);
		}
		if (++count > config.THREADS_PER_PAGE) {
			/* Sync logic isn't great yet; allow this for now */
			// return false;
		}
	}
	client.watching = syncs;
	if (live) {
		/* XXX: This will break if a thread disappears during sync
		 *      (won't be reported)
		 * Or if any of the threads they see on the first page
		 * don't show up in the 'live' pub for whatever reason.
		 * Really we should get them synced first and *then* switch
		 * to the live pub.
		 */
		client.watching = {live: true};
		count = 1;
	}
	/* Race between subscribe and backlog fetch; client must de-dup */
	client.db.kiku(client.watching, client.on_update.bind(client),
			client.on_thread_sink.bind(client), listening);
	function listening(errs) {
		if (errs && errs.length >= count)
			return report("Couldn't sync to board.", client);
		else if (errs) {
			dead_threads.push.apply(dead_threads, errs);
			errs.forEach(function (thread) {
				delete client.watching[thread];
			});
		}
		client.db.fetch_backlogs(client.watching, got_backlogs);
	}
	function got_backlogs(errs, logs) {
		if (errs) {
			dead_threads.push.apply(dead_threads, errs);
			errs.forEach(function (thread) {
				delete client.watching[thread];
			});
		}

		logs.push([common.SYNCHRONIZE, dead_threads]);
		client.socket.send(JSON.stringify(logs));
		client.synced = true;
	}
	return true;
}

var unskippable = [common.FINISH_POST, common.DELETE_POSTS,
		common.DELETE_THREAD];

OK.on_update = function(op, num, kind, msg) {
	var mine = (this.post && this.post.num == num) || this.last_num == num;
	if (mine && unskippable.indexOf(kind) < 0) {
		this.skipped++;
		return;
	}
	else if (this.post && kind == common.DELETE_POSTS) {
		/* grr special case */
		var nums = JSON.parse('[' + msg + ']');
		if (nums.indexOf(this.post.num) >= 0) {
			this.last_num = this.post.num;
			delete this.post;
		}
	}
	else if (this.post && kind == common.DELETE_THREAD) {
		/* GRR */
		if (this.post.num == op || this.post.op == op) {
			this.last_num = this.post.num;
			delete this.post;
		}
	}
	msg = '[' + msg + ',' + op + ']';
	if (this.skipped) {
		var skipped_op = this.post ? (this.post.op || this.post.num)
				: db.OPs[this.last_num];
		var catch_up = [common.CATCH_UP, skipped_op, this.skipped];
		msg = JSON.stringify(catch_up) + ',' + msg;
		this.skipped = 0;
	}
	this.socket.send('[' + msg + ']');
};

OK.on_thread_sink = function (thread, err) {
	/* TODO */
	console.log(thread, 'sank:', err);
};

var oneeSama = new common.OneeSama(function (num) {
	var op = db.OPs[num];
	if (op)
		this.callback(common.safe('<a href="'
				+ common.post_url({op: op, num: num}, false)
				+ '">&gt;&gt;' + num + '</a>'));
	else
		this.callback('>>' + num);
});
oneeSama.media = config.MEDIA_URL;

function write_thread_html(reader, response, full_thread) {
	reader.on('thread', function (op_post, omit, image_omit) {
		oneeSama.full = full_thread;
		var first = oneeSama.monomono(op_post);
		first.pop();
		response.write(first.join(''));
		if (omit)
			response.write('\t<span class="omit">' +
				common.abbrev_msg(omit, image_omit) +
				'</span>\n');
	});
	reader.on('post', function (post) {
		oneeSama.full = full_thread;
		response.write(oneeSama.mono(post));
	});
	reader.on('endthread', function () {
		response.write('</section><hr>\n');
	});
}

function image_status(status) {
	this.client.send([common.IMAGE_STATUS, status]);
}

function page_nav(thread_count, cur_page) {
	var page_count = Math.ceil(thread_count / config.THREADS_PER_PAGE);
	page_count = Math.max(page_count, 1);
	var info = {pages: page_count, threads: thread_count,
		cur_page: cur_page};
	var next = Math.max(cur_page, 0) + 1;
	if (next < page_count)
		info.next_page = 'page' + next;
	var prev = cur_page - 1;
	if (prev >= 0)
		info.prev_page = 'page' + prev;
	return info;
}

function make_board_meta(info) {
	var bits = [];
	if (info.cur_page >= 0)
		bits.push(['index', 'live']);
	if (info.prev_page)
		bits.push(['prev', info.prev_page]);
	if (info.next_page)
		bits.push(['next', info.next_page]);
	return bits.map(function (p) {
		return '\t<link rel="'+p[0]+'" href="'+p[1]+'">\n';
	}).join('');
}

function make_nav_html(info) {
	var bits = ['<nav>'], cur = info.cur_page;
	if (cur >= 0)
		bits.push('<a href="live">live</a>');
	else
		bits.push('<b>live</b>');
	for (var i = 0; i < info.pages; i++) {
		if (i != cur)
			bits.push('<a href="page' + i + '">' + i + '</a>');
		else
			bits.push('<b>' + i + '</b>');
	}
	if (info.next_page)
		bits.push(' <input type="button" value="Next">'); // TODO
	bits.push('</nav>');
	return bits.join('');
}

var server = http.createServer(function (req, resp) {
	var method = req.method.toLowerCase(), numRoutes = routes.length;
	var parsed = url_parse(req.url, true);
	req.url = parsed.pathname;
	req.query = parsed.query;
	for (var i = 0; i < numRoutes; i++) {
		var route = routes[i];
		if (method != route.method)
			continue;
		var m = req.url.match(route.pattern);
		if (m) {
			route.handler(req, resp, m);
			return;
		}
	}
	if (debug_static)
		debug_static(req, resp);
	else
		render_404(resp);
});

var routes = [];

function route_get(pattern, handler) {
	routes.push({method: 'get', pattern: pattern,
			handler: auth_passthrough.bind(null, handler)});
}

function auth_passthrough(handler, req, resp, params) {
	if (!twitter.check_cookie(req.headers.cookie, false, go))
		handler(req, resp, params);

	function go(err, session) {
		if (!err)
			req.auth = session;
		handler(req, resp, params);
	}
}

function route_get_auth(pattern, handler) {
	routes.push({method: 'get', pattern: pattern,
			handler: auth_checker.bind(null, handler, false)});
}

function parse_post_body(req, callback) {
	// jesus christ
	var buf = [], len = 0;
	req.on('data', function (data) {
		buf.push(data);
		len += data.length;
	});
	req.once('end', function () {
		var i = 0;
		var dest = new Buffer(len);
		buf.forEach(function (b) {
			b.copy(dest, i, 0);
			i += b.length;
		});
		var combined = dest.toString('utf-8');
		var body = {};
		combined.split('&').forEach(function (param) {
			var m = param.match(/^(.*?)=(.*)$/);
			if (m)
				body[decodeURIComponent(m[1])] = (
					decodeURIComponent(m[2]));
		});
		buf = dest = combined = null;
		callback(null, body);
	});
	req.once('close', callback);
}

function auth_checker(handler, is_post, req, resp, params) {
	if (is_post) {
		parse_post_body(req, function (err, body) {
			if (err) {
				resp.writeHead(500, noCacheHeaders);
				resp.end(preamble + escape(err));
				return;
			}
			req.body = body;
			check_it();
		});
	}
	else
		check_it();

	function check_it() {
		if (!twitter.check_cookie(req.headers.cookie, is_post, ack))
			return forbidden('No cookie.');
	}

	function ack(err, session) {
		if (err)
			return forbidden(err);
		req.auth = session;
		handler(req, resp, params);
	}

	function forbidden(err) {
		resp.writeHead(401, noCacheHeaders);
		resp.end(preamble + escape(err));
	}
}

function route_post_auth(pattern, handler) {
	routes.push({method: 'post', pattern: pattern,
			handler: auth_checker.bind(null, handler, true)});
}


var debug_static = !config.DEBUG ? false : function (req, resp) {
	/* Highly insecure. */
	var url = req.url.replace(/\.\.+/g, '');
	var path = require('path').join(__dirname, '..', 'www', url);
	var s = fs.createReadStream(path);
	s.once('error', function (err) {
		if (err.code == 'ENOENT')
			render_404(resp);
		else {
			resp.writeHead(500, noCacheHeaders);
			resp.end(preamble + escape(err.message));
		}
	});
	s.once('open', function () {
		var h = {};
		try {
			var mime = require('connect').utils.mime;
			var ext = require('path').extname(path);
			h['Content-Type'] = mime.type(ext);
		} catch (e) {}
		resp.writeHead(200, h);
		util.pump(s, resp);
	});
	return true;
};

var vanillaHeaders = {'Content-Type': 'text/html; charset=UTF-8'};
var noCacheHeaders = {'Content-Type': 'text/html; charset=UTF-8',
		'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT, -1',
		'Cache-Control': 'no-cache'};
var preamble = '<!doctype html><meta charset=utf-8>';

function render_404(resp) {
	resp.writeHead(404, noCacheHeaders);
	resp.end(notFoundHtml);
}

function redirect(resp, uri, code) {
	var headers = {Location: uri};
	for (var k in vanillaHeaders)
		headers[k] = vanillaHeaders[k];
	resp.writeHead(code || 303, headers);
	resp.end(preamble + '<title>Redirect</title>'
		+ '<a href="' + encodeURI(uri) + '">Proceed</a>.');
}

function redirect_thread(resp, num, op) {
	redirect(resp, op + '#' + num);
}

routes.push({method: 'post', pattern: /^\/img$/, handler: function (req,resp) {
	var upload = new pix.ImageUpload(clients, allocate_post, image_status);
	upload.handle_request(req, resp);
}});

route_get(/^\/$/, function (req, resp) {
	redirect(resp, 'moe/');
});

if (config.DEBUG) {
	route_get(/^\/login$/, function (req, resp) {
		twitter.set_cookie(resp);
	});
}
else {
	route_get(/^\/login$/, twitter.login);
	route_get(/^\/verify$/, twitter.verify);
}

var filterTmpl;
route_get_auth(/^\/admin$/, function (req, resp) {
	var who = req.auth.user || 'unknown';

	var img = _.template('<a href="moe/{{num}}">'
			+ '<img alt="{{num}}" title="Thread {{num}}" src="'
			+ config.MEDIA_URL + 'thumb/{{thumb}}" width=50 '
			+ 'height=50></a>\n');
	var limit = parseInt(req.query.limit, 10) || 0;
	var ctr = 0;

	resp.writeHead(200);
	resp.write(filterTmpl[0]);
	resp.write('<h2>Limit ' + limit + '</h2>\n');

	var filter = new db.Filter('moe');
	filter.get_all(limit);

	filter.on('thread', function (thread) {
		resp.write(img(thread));
		ctr += 1;
	});
	filter.once('end', function () {
		resp.write('<br>' + ctr + ' thread(s).');
		resp.end(filterTmpl[1]);
	});
	filter.once('error', function (err) {
		resp.end('<br><br>Error: ' + escape(err));
	});
});

route_post_auth(/^\/admin$/, function (req, resp) {

	var threads = req.body.threads.split(',').map(function (x) {
		return parseInt(x, 10);
	}).filter(function (x) {
		return !isNaN(x);
	});

	var yaku = new db.Yakusoku;
	yaku.remove_posts(threads, function (err, dels) {

		// XXX: Can't disconnect right away.
		//      Does its business in the background.
		//      Grrr. Hack for now.
		setTimeout(function () {
			yaku.disconnect();
		}, 30 * 1000);

		if (err) {
			resp.writeHead(500, noCacheHeaders);
			resp.end(preamble + escape(err));
			return;
		}
		resp.writeHead(200, noCacheHeaders);
		resp.end();
	});
});

route_get_auth(/^\/admin\.js$/, function (req, resp, params) {
	resp.writeHead(200, {'Content-Type': 'text/javascript'});
	if (config.DEBUG)
		resp.end(fs.readFileSync('admin.js'));
	else
		resp.end(adminJs);
});

route_get(/^\/(\w+)$/, function (req, resp, params) {
	redirect(resp, params[1] + '/live');
});
route_get(/^\/\w+\/$/, function (req, resp) {
	redirect(resp, 'live');
});

route_get(/^\/(\w+)\/live$/, function (req, resp, params) {
	if (params[1] != 'moe') // TEMP
		return render_404(resp);
	var yaku = new db.Yakusoku();
	yaku.get_tag(0);
	var nav_html;
	yaku.on('begin', function (thread_count) {
		var nav = page_nav(thread_count, -1);
		resp.writeHead(200, noCacheHeaders);
		resp.write(indexTmpl[0]);
		resp.write(make_board_meta(nav));
		resp.write(indexTmpl[1]);
		resp.write(config.TITLE);
		resp.write(indexTmpl[2]);
		nav_html = make_nav_html(nav);
		resp.write(nav_html);
		resp.write('<hr>\n');
	});
	write_thread_html(yaku, resp, false);
	yaku.on('end', function () {
		resp.write(nav_html);
		write_page_end(req, resp);
		yaku.disconnect();
	});
	yaku.on('error', function (err) {
		console.error('index:', err);
		resp.end();
		yaku.disconnect();
	});
	return true;
});
route_get(/^\/\w+\/live\/$/, function (req, resp, params) {
	redirect(resp, '../live');
});

route_get(/^\/(\w+)\/page(\d+)$/, function (req, resp, params) {
	if (params[1] != 'moe') // TEMP
		return render_404(resp);
	var yaku = new db.Yakusoku();
	var page = parseInt(params[2], 10);
	if (page > 0 && params[2][0] == '0') /* leading zeroes? */
		return redirect(resp, 'page' + page);
	yaku.get_tag(page);
	yaku.on('nomatch', render_404.bind(null, resp));
	var nav_html;
	yaku.on('begin', function (thread_count) {
		var nav = page_nav(thread_count, page);
		resp.writeHead(200, noCacheHeaders);
		resp.write(indexTmpl[0]);
		resp.write(make_board_meta(nav));
		resp.write(indexTmpl[1]);
		resp.write(config.TITLE);
		resp.write(indexTmpl[2]);
		nav_html = make_nav_html(nav);
		resp.write(nav_html);
		resp.write('<hr>\n');
	});
	write_thread_html(yaku, resp, false);
	yaku.on('end', function () {
		resp.write(nav_html);
		write_page_end(req, resp);
		yaku.disconnect();
	});
	yaku.on('error', function (err) {
		console.error('page', page + ':', err);
		resp.end();
		yaku.disconnect();
	});
	return true;
});
route_get(/^\/\w+\/page(\d+)\/$/, function (req, resp, params) {
	redirect(resp, '../page' + params[1]);
});

route_get(/^\/(\w+)\/(\d+)$/, function (req, resp, params) {
	if (params[1] != 'moe') // TEMP
		return render_404(resp);
	var num = parseInt(params[2], 10);
	if (!num)
		return render_404(resp);
	else if (params[2][0] == '0')
		return redirect(resp, '' + num);
	var op = db.OPs[num];
	if (!op)
		return render_404(resp);
	if (op != num)
		return redirect_thread(resp, num, op);
	var yaku = new db.Yakusoku();
	var reader = new db.Reader(yaku);
	reader.get_thread(num, true, false);
	reader.on('nomatch', render_404.bind(null, resp));
	reader.on('redirect', redirect_thread.bind(null, resp, num));
	reader.on('begin', function () {
		resp.writeHead(200, noCacheHeaders);
		resp.write(indexTmpl[0]);
		resp.write('\t<link rel="index" href="live">\n');
		resp.write(indexTmpl[1]);
		resp.write('Thread #' + op);
		resp.write(indexTmpl[2]);
		resp.write('<hr>\n');
	});
	write_thread_html(reader, resp, true);
	reader.on('end', function () {
		resp.write('[<a href=".">Return</a>]');
		write_page_end(req, resp);
		yaku.disconnect();
	});
	function on_err(err) {
		console.error('thread '+num+':', err);
		resp.end();
		yaku.disconnect();
	}
	reader.on('error', on_err);
	yaku.on('error', on_err);
	return true;
});
route_get(/^\/\w+\/(\d+)\/$/, function (req, resp, params) {
	redirect(resp, '../' + params[1]);
});

function write_page_end(req, resp) {
	resp.write(indexTmpl[3]);
	if (req.auth)
		resp.write('<script src="../admin.js"></script>\n');
	resp.end();
}

OK.on_message = function (data) {
	var msg;
	try { msg = JSON.parse(data); }
	catch (e) {}
	var type = common.INVALID;
	if (msg) {
		if (this.post && typeof msg == 'string')
			type = common.UPDATE_POST;
		else if (msg.constructor == Array)
			type = msg.shift();
	}
	var func = dispatcher[type];
	if (!func || !func(msg, this)) {
		console.error("Got invalid message " + data);
		report(null, this, "Bad protocol.");
	}
};

OK.on_disconnect = function () {
	delete clients[this.id];
	this.synced = false;
	var db = this.db;
	if (this.watching)
		db.kikanai(this.watching);
	if (this.post)
		this.finish_post(function () {
			db.disconnect();
		});
	else
		db.disconnect();
};

function pad3(n) {
	return (n < 10 ? '00' : (n < 100 ? '0' : '')) + n;
}

var git_version;
var error_db;
function report(error, client, client_msg) {
	if (typeof git_version == 'undefined') {
		git_version = null;
		get_version([], function (err, ver) {
			if (err) {
				console.error(err);
				console.error(error);
			}
			else {
				git_version = ver;
				report(error, client, client_msg);
			}
		});
		return;
	}
	if (!error_db)
		error_db = new db.Yakusoku;
	var ver = git_version || 'ffffff';
	var msg = client_msg || 'Server error.';
	var ip = client && client.ip;
	var info = {error: error, msg: msg, ip: ip};
	error_db.report_error(info, ver, function (err, num) {
		if (err)
			console.error(err);
		ver = ' (#' + ver + '-' + pad3(num) + ')';
		console.error((error || msg) + ' ' + ip + ver);
		if (client) {
			client.send([common.INVALID, msg + ver]);
			client.synced = false;
		}
	});
}

/* Must be prepared to receive callback instantly */
function valid_links(frag, state, callback) {
	var links = {};
	var onee = new common.OneeSama(function (num) {
		if (num in db.OPs)
			links[num] = db.OPs[num];
	});
	onee.callback = function (frag) {};
	onee.state = state;
	onee.fragment(frag);
	callback(null, common.is_empty(links) ? null : links);
}

dispatcher[common.ALLOCATE_POST] = function (msg, client) {
	if (msg.length != 1)
		return false;
	msg = msg[0];
	if (!msg || typeof msg != 'object' || !msg.op)
		return false;
	if (client.post)
		return update_post(msg.frag, client);
	var frag = msg.frag;
	if (!frag || frag.match(/^\s*$/g))
		return false;
	allocate_post(msg, null, client, function (err, alloc) {
		if (err)
			return report(err, client, "Couldn't allocate post.");
		var go = client.send.bind(client,
				[common.ALLOCATE_POST, alloc]);
		if (!config.DEBUG)
			go();
		else
			setTimeout(go, 500);
	});
	return true;
}

function allocate_post(msg, image, client, callback) {
	if (!msg || typeof msg != 'object')
		return callback('Bad alloc.');
	if (client.post)
		return callback("Already have a post.");
	var post = {time: new Date().getTime()};
	var body = '';
	if (msg.frag !== undefined) {
		if (typeof msg.frag != 'string' || msg.frag.match(/^\s*$/g)
				|| msg.frag.length > common.MAX_POST_CHARS)
			return callback('Post is too long.');
		body = msg.frag;
	}
	if (msg.op !== undefined) {
		if (typeof msg.op != 'number')
			return callback('Invalid thread.');
		post.op = msg.op;
	}
	/* TODO: Check against client.watching? */
	if (msg.name !== undefined) {
		if (typeof msg.name != 'string')
			return callback('Invalid name.');
		var parsed = common.parse_name(msg.name);
		post.name = parsed[0];
		if (parsed[1] || parsed[2]) {
			var trip = tripcode.hash(parsed[1], parsed[2]);
			if (trip)
				post.trip = trip;
		}
	}
	if (msg.email !== undefined) {
		if (typeof msg.email != 'string')
			return callback('Invalid email.');
		post.email = msg.email.trim().substr(0, 320);
		if (common.is_noko(post.email))
			delete post.email;
	}
	if (image)
		post.image = image;
	post.state = [0, 0];

	client.db.reserve_post(post.op, got_reservation);
	function got_reservation(err, num) {
		if (err)
			return callback("Couldn't reserve a post.");
		if (client.post)
			return callback('Already have a post.');
		client.post = post;
		post.num = num;
		valid_links(body, post.state, got_links);
	}
	function got_links(err, links) {
		if (err) {
			console.error('valid_links: ' + err);
			if (client.post === post)
				delete client.post;
			return callback("Post reference error.");
		}
		post.links = links;
		client.db.insert_post(post, body, client.ip, inserted);
	}
	function inserted(err) {
		if (err) {
			if (client.post === post)
				delete client.post;
			console.error(err);
			return callback("Couldn't allocate post.");
		}
		post.body = body;
		callback(null, get_post_view(post));
	}
	return true;
}

function get_post_view(post) {
	var view = {num: post.num, body: post.body, time: post.time};
	if (post.op) view.op = post.op;
	if (post.name) view.name = post.name;
	if (post.trip) view.trip = post.trip;
	if (post.email) view.email = post.email;
	if (post.editing) view.editing = post.editing;
	if (post.links) view.links = post.links;
	if (post.image) view.image = post.image;
	return view;
}

function update_post(frag, client) {
	if (typeof frag != 'string')
		return false;
	var post = client.post;
	if (!post)
		return false;
	var limit = common.MAX_POST_CHARS;
	if (frag.length > limit || post.length >= limit)
		return false;
	var combined = post.length + frag.length;
	if (combined > limit)
		frag = frag.substr(0, combined - limit);
	post.body += frag;
	/* imporant: broadcast prior state */
	var old_state = post.state.slice();

	valid_links(frag, post.state, function (err, links) {
		if (err)
			links = null; /* oh well */
		var new_links = {};
		if (links) {
			if (!post.links)
				post.links = {};
			for (var k in links) {
				var link = links[k];
				if (post.links[k] != link) {
					post.links[k] = link;
					new_links[k] = link;
				}
			}
		}

		client.db.append_post(post, frag, old_state, links, new_links,
				function (err) {
			if (err)
				report(err, client, "Couldn't add text.");
		});
	});
	return true;
}
dispatcher[common.UPDATE_POST] = update_post;

OK.finish_post = function (callback) {
	/* TODO: Should we check this.uploading? */
	var self = this;
	this.db.finish_post(this.post, function (err) {
		if (err)
			callback(err);
		else {
			self.last_num = self.post.num;
			delete self.post;
			callback(null);
		}
	});
}

dispatcher[common.FINISH_POST] = function (msg, client) {
	if (msg.length || !client.post)
		return false;
	client.finish_post(function (err) {
		if (err)
			report(err, client, "Couldn't finish post.");
	});
	return true;
}

function auth_handled(func) {
	return function (msg, client) {
		if (!msg.length || !twitter.check_cookie(msg.shift(),false,go))
			return false;
		function go(err, session) {
			if (err || common.is_empty(session))
				report(err, client, 'Auth error.');
			else
				func(msg, client);
		}
		return true;
	};
}

dispatcher[common.DELETE_POSTS] = auth_handled(function (nums, client) {
	if (!nums.length)
		return false;
	if (nums.some(function (n) { return typeof n != 'number' || n < 1; }))
		return false;

	/* Omit to-be-deleted posts that are inside to-be-deleted threads */
	var ops = {}, OPs = db.OPs;
	nums.forEach(function (num) {
		if (num == OPs[num])
			ops[num] = 1;
	});
	nums = nums.filter(function (num) {
		var op = OPs[num];
		return op == num || !(OPs[num] in ops);
	});

	client.db.remove_posts(nums, function (err, dels) {
		if (err)
			report(err, client, "Couldn't delete.");
	});
	return true;
});

function start_server() {
	server.listen(config.PORT);
	var io = require('socket.io').listen(server, {
		heartbeats: !config.DEBUG,
		'log level': config.DEBUG ? 2 : 1,
		'flash policy server': false,
		'browser client': false,
	});
	io.sockets.on('connection', function on_client (socket) {
		var client = new Okyaku(socket);
		clients[client.id] = client;
	});
	io.sockets.on('error', function (err) {
		console.log(err);
	});
}

function get_version(deps, callback) {
	require('child_process').exec('git log -1 --format=%h '+deps.join(' '),
			function (err, stdout, stderr) {
		if (err)
			callback(err);
		else
			callback(null, stdout.trim());
	});
}

(function () {

if (process.argv[2] == '--show-config') {
	var key = process.argv[3];
	if (!(key in config))
		throw "No such config value " + process.argv[3];
	var val = config[process.argv[3]];
	console.log((val && val.join) ? val.join(' ') : val);
}
else if (process.argv[2] == '--client-version')
	get_version(config.CLIENT_DEPS, function (err, version) {
		if (err)
			throw err;
		else
			console.log(version);
	});
else {
	get_version(config.CLIENT_DEPS, function (err, version) {
		if (err)
			throw err;
		tripcode = require('./tripcode');
		config.CLIENT_JS = 'client-' + version + (
				config.DEBUG ? '.debug.js' : '.js');
		indexTmpl = _.template(fs.readFileSync('index.html', 'UTF-8'),
				config).split(/\$[A-Z]+/);
		filterTmpl = _.template(fs.readFileSync('filter.html', 'UTF-8'),
				config).split(/\$[A-Z]+/);
		notFoundHtml = fs.readFileSync('../www/404.html');
		adminJs = fs.readFileSync('admin.js');
		db.track_OPs(function (err) {
			if (err)
				throw err;
			var yaku = new db.Yakusoku;
			yaku.finish_all(function (err) {
				if (err)
					throw err;
				yaku.disconnect();
				setTimeout(start_server, 0);
			});
		});
	});
}

})();
