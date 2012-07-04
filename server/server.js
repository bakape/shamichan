var _ = require('../lib/underscore'),
    amusement = require('./amusement'),
    async = require('async'),
    caps = require('./caps'),
    common = require('../common'),
    config = require('../config'),
    db = require('../db'),
    fs = require('fs'),
    get_version = require('../get').get_version,
    hooks = require('../hooks'),
    pix = require('./pix'),
    STATE = require('./state');
    twitter = require('./twitter'),
    tripcode = require('./tripcode'),
    web = require('./web'),
    winston = require('winston');

require('./panel');

var RES = STATE.resources;

var clients = {};
var dispatcher = {};

/* I always use encodeURI anyway */
var escape = common.escape_html;
var safe = common.safe;

function Okyaku(socket, ip) {
	this.socket = socket;
	this.ident = caps.lookup_ident(ip);
	this.watching = {};
}
var OK = Okyaku.prototype;

OK.send = function (msg) {
	this.socket.write(JSON.stringify([msg]));
};

dispatcher[common.SYNCHRONIZE] = function (msg, client) {
	function checked(err, ident) {
		if (!err)
			_.extend(client.ident, ident);
		if (!synchronize(msg, client))
			report("Bad protocol.", client);
	}
	var chunks = twitter.extract_cookie(msg.pop());
	if (chunks) {
		twitter.check_cookie(chunks, false, checked);
		return true;
	}
	else
		return synchronize(msg, client);
};

function synchronize(msg, client) {
	if (msg.length != 4)
		return false;
	var id = msg[0], board = msg[1], syncs = msg[2], live = msg[3];
	if (!id || typeof id != 'number' || id < 0 || Math.round(id) != id)
		return false;
	if (id in clients) {
		winston.error("Duplicate client id " + id);
		return false;
	}
	client.id = id;
	clients[id] = client;
	if (!syncs || typeof syncs != 'object')
		return false;
	if (client.synced) {
		//winston.warn("Client tried to sync twice");
		/* Sync logic is buggy; allow for now */
		//return true;
	}
	if (!board || !caps.can_access(client.ident, board))
		return false;
	var dead_threads = [], count = 0, op;
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
		op = k;
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
	client.board = board;

	if (client.db)
		client.db.disconnect();
	client.db = new db.Yakusoku(board, client.ident);
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

		var sync = '0,' + common.SYNCHRONIZE;
		if (dead_threads.length)
			sync += ',' + JSON.stringify(dead_threads);
		logs.push(sync);
		client.socket.write('[[' + logs.join('],[') + ']]');
		client.synced = true;

		var info = {client: client, live: live, count: count, op: op};
		hooks.trigger('clientSynced', info, function (err) {
			if (err)
				winston.error(err);
		});
	}
	return true;
}

OK.on_update = function (op, kind, msg) {
	// Special cases for operations that overwrite a client's state
	if (this.post && kind == common.DELETE_POSTS) {
		var nums = JSON.parse(msg)[0].slice(2);
		if (nums.indexOf(this.post.num) >= 0)
			this.post = null;
	}
	else if (this.post && kind == common.DELETE_THREAD) {
		if (this.post.num == op || this.post.op == op)
			this.post = null;
	}

	this.socket.write(msg);
};

OK.on_thread_sink = function (thread, err) {
	/* TODO */
	winston.error(thread, 'sank:', err);
};

function tamashii(num) {
	var op = db.OPs[num];
	if (op)
		this.callback(this.post_ref(num, op));
	else
		this.callback('>>' + num);
}

function write_thread_html(reader, response, opts) {
	var oneeSama = new common.OneeSama(tamashii);
	caps.augment_oneesama(oneeSama, opts);
	reader.on('thread', function (op_post, omit, image_omit) {
		op_post.omit = omit;
		var full = oneeSama.full = !!opts.fullPosts;
		oneeSama.op = opts.fullLinks ? false : op_post.num;
		var first = oneeSama.monomono(op_post, full && 'full');
		first.pop();
		response.write(first.join(''));
		if (omit)
			response.write('\t<span class="omit">' +
				common.abbrev_msg(omit, image_omit) +
				'</span>\n');
	});
	reader.on('post', function (post) {
		response.write(oneeSama.mono(post));
	});
	reader.on('endthread', function () {
		response.write('</section><hr>\n');
	});
}

function image_status(client_id, status) {
	var client = clients[client_id];
	if (client)
		client.send([0, common.IMAGE_STATUS, status]);
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

function make_link_rels(board, bits) {
	bits.push(['stylesheet', config.MEDIA_URL + STATE.hot.BASE_CSS]);
	bits.push(['stylesheet', config.MEDIA_URL +
			STATE.hot.BOARD_CSS[board], 'theme']);
	return bits.map(function (p) {
		var html = '\t<link rel="'+p[0]+'" href="'+p[1]+'"';
		if (p[2])
			html += ' id="' + p[2] + '"';
		return html + '>\n';
	}).join('');
}

function make_board_meta(board, info) {
	var bits = [];
	if (info.cur_page >= 0)
		bits.push(['index', 'live']);
	if (info.prev_page)
		bits.push(['prev', info.prev_page]);
	if (info.next_page)
		bits.push(['next', info.next_page]);
	return make_link_rels(board, bits);
}

function make_thread_meta(board, num, abbrev) {
	var bits = [['index', 'live']];
	if (abbrev)
		bits.push(['canonical', num]);
	return make_link_rels(board, bits);
}

function make_nav_html(info) {
	var bits = ['<nav>'], cur = info.cur_page;
	if (cur >= 0)
		bits.push('<a href=".">live</a>');
	else
		bits.push('<strong>live</strong>');
	for (var i = 0; i < info.pages; i++) {
		if (i != cur)
			bits.push('<a href="page' + i + '">' + i + '</a>');
		else
			bits.push('<strong>' + i + '</strong>');
	}
	if (info.next_page)
		bits.push(' <input type="button" value="Next">'); // TODO
	bits.push('</nav>');
	return bits.join('');
}

function redirect_thread(resp, num, op, tag) {
	if (!tag)
		web.redirect(resp, op + '#' + num);
	else
		/* Use a JS redirect to preserve the hash */
		web.redirect_js(resp, '../' + tag + '/' + op + '#' + num);
}

web.route_post(/^\/(\w+)\/upload$/, function (req, resp, params) {
	var board = params[1];
	if (!caps.can_access(req.ident, board))
		return web.render_404(resp);
	var yaku = new db.Yakusoku(board, req.ident);
	var upload = new pix.ImageUpload(yaku, image_status);
	upload.handle_request(req, resp, board);
});

web.route_get(/^\/$/, function (req, resp) {
	web.redirect(resp, 'moe/');
});

if (config.DEBUG) {
	web.route_get(/^\/login$/, function (req, resp) {
		twitter.set_cookie(resp, {auth: 'Admin'});
	});
	web.route_get(/^\/mod$/, function (req, resp) {
		twitter.set_cookie(resp, {auth: 'Moderator'});
	});
}
else {
	web.route_get(/^\/login$/, twitter.login);
	web.route_get(/^\/verify$/, twitter.verify);
}

web.route_get(/^\/login\/$/, function (req, resp) {
	web.redirect(resp, '../login');
});

web.route_post(/^\/logout$/, twitter.logout);
if (config.DEBUG) {
	web.route_get(/^\/logout$/, twitter.logout);
	web.route_get(/^\/logout\/$/, function (req, resp) {
		web.redirect(resp, '../logout');
	});
}

function write_mod_js(resp, ident) {
	resp.writeHead(200, {
			'Content-Type': 'text/javascript; charset=UTF-8'});
	resp.write('(function (AUTH) {');
	resp.write(RES.modJs);
	resp.end('})(' + JSON.stringify(ident) + ');');
}

web.route_get_auth(/^\/admin\.js$/, function (req, resp, params) {
	if (req.ident.auth != 'Admin')
		return web.render_404(resp);
	write_mod_js(resp, 'Admin');
});

web.route_get_auth(/^\/mod\.js$/, function (req, resp, params) {
	if (req.ident.auth != 'Moderator')
		return web.render_404(resp);
	write_mod_js(resp, 'Moderator');
});

web.route_get(/^\/(\w+)$/, function (req, resp, params) {
	var board = params[1];
	/* If arbitrary boards were allowed, need to escape this: */
	var dest = board + '/';
	if (caps.under_curfew(req.ident, board))
		return web.redirect(resp, dest);
	if (!caps.can_access(req.ident, board))
		return web.render_404(resp);
	web.redirect(resp, dest);
});

web.route_get(/^\/(\w+)\/live$/, function (req, resp, params) {
	var board = params[1];
	if (caps.under_curfew(req.ident, board))
		return web.redirect(resp, '.');
	if (!caps.can_access(req.ident, board))
		return web.render_404(resp);
	web.redirect(resp, '.');
});

web.route_get(/^\/(\w+)\/$/, function (req, resp, params) {
	var board = params[1];
	if (caps.under_curfew(req.ident, board)) {
		resp.writeHead(200, web.noCacheHeaders);
		resp.write(RES.curfewTmpl[0]);
		resp.write('/' + board + '/');
		resp.write(RES.curfewTmpl[1]);
		var ending = caps.curfew_ending_time(board);
		resp.write(ending ? ''+ending.getTime() : 'null');
		resp.end(RES.curfewTmpl[2]);
		return;
	}
	if (!caps.can_access(req.ident, board))
		return web.render_404(resp);
	var yaku = new db.Yakusoku(board, req.ident);
	yaku.get_tag(0);
	var indexTmpl = RES.indexTmpl, nav_html;
	yaku.on('begin', function (thread_count) {
		var nav = page_nav(thread_count, -1);
		resp.writeHead(200, web.noCacheHeaders);
		var title = STATE.hot.TITLES[board] || escape(board);
		resp.write(indexTmpl[0]);
		resp.write(title);
		resp.write(indexTmpl[1]);
		resp.write(make_board_meta(board, nav));
		resp.write(indexTmpl[2]);
		resp.write(title);
		resp.write(indexTmpl[3]);
		nav_html = make_nav_html(nav);
		resp.write(nav_html);
		resp.write('<hr>\n');
	});
	var opts = {fullLinks: true, ident: req.ident, board: board};
	write_thread_html(yaku, resp, opts);
	yaku.on('end', function () {
		resp.write(nav_html);
		write_page_end(req, resp);
		yaku.disconnect();
	});
	yaku.on('error', function (err) {
		winston.error('index:', err);
		resp.end();
		yaku.disconnect();
	});
	return true;
});

web.route_get(/^\/(\w+)\/page(\d+)$/, function (req, resp, params) {
	var board = params[1];
	if (caps.under_curfew(req.ident, board))
		return web.redirect(resp, '..', 302);
	if (!caps.can_access(req.ident, board))
		return web.render_404(resp);
	var yaku = new db.Yakusoku(board, req.ident);
	var page = parseInt(params[2], 10);
	if (page > 0 && params[2][0] == '0') /* leading zeroes? */
		return web.redirect(resp, 'page' + page);
	yaku.get_tag(page);
	yaku.on('nomatch', web.render_404.bind(null, resp));
	var indexTmpl = RES.indexTmpl, nav_html;
	yaku.on('begin', function (thread_count) {
		var nav = page_nav(thread_count, page);
		resp.writeHead(200, web.noCacheHeaders);
		var title = STATE.hot.TITLES[board] || escape(board);
		resp.write(indexTmpl[0]);
		resp.write(title);
		resp.write(indexTmpl[1]);
		resp.write(make_board_meta(board, nav));
		resp.write(indexTmpl[2]);
		resp.write(title);
		resp.write(indexTmpl[3]);
		nav_html = make_nav_html(nav);
		resp.write(nav_html);
		resp.write('<hr>\n');
	});
	var opts = {fullLinks: true, ident: req.ident, board: board};
	write_thread_html(yaku, resp, opts);
	yaku.on('end', function () {
		resp.write(nav_html);
		write_page_end(req, resp);
		yaku.disconnect();
	});
	yaku.on('error', function (err) {
		winston.error('page', page + ':', err);
		resp.end();
		yaku.disconnect();
	});
	return true;
});
web.route_get(/^\/(\w+)\/page(\d+)\/$/, function (req, resp, params) {
	var board = params[1];
	if (caps.under_curfew(req.ident, board))
		return web.redirect(resp, '..', 302);
	web.redirect(resp, '../page' + params[2]);
});

var returnHTML = '<span id="return" class="act"><a href=".">Return</a></span>';

web.route_get(/^\/(\w+)\/(\d+)$/, function (req, resp, params) {
	var board = params[1];
	if (caps.under_curfew(req.ident, board))
		return web.redirect(resp, '.', 302);
	if (!caps.can_access(req.ident, board))
		return web.render_404(resp);
	var num = parseInt(params[2], 10);
	if (!num)
		return web.render_404(resp);
	else if (params[2][0] == '0')
		return web.redirect(resp, '' + num);
	var op;
	if (board == 'graveyard') {
		op = num;
	}
	else {
		op = db.OPs[num];
		if (!op)
			return web.render_404(resp);
		if (!db.OP_has_tag(board, op)) {
			var tag = db.first_tag_of(op);
			if (tag)
				return redirect_thread(resp, num, op, tag);
			else {
				winston.warn("Orphaned post", num,
					"with tagless OP", op);
				return web.render_404(resp);
			}
		}
		if (op != num)
			return redirect_thread(resp, num, op);
	}
	var yaku = new db.Yakusoku(board, req.ident);
	var reader = new db.Reader(yaku);
	var lastN = config.THREAD_LAST_N;
	var limit = ('last' + lastN) in req.query ?
			(lastN + config.ABBREVIATED_REPLIES) : 0;
	reader.get_thread(board, num, {redirect: true, abbrev: limit});
	reader.on('nomatch', web.render_404.bind(null, resp));
	reader.on('redirect', redirect_thread.bind(null, resp, num));
	reader.on('begin', function () {
		var indexTmpl = RES.indexTmpl;
		resp.writeHead(200, web.noCacheHeaders);
		resp.write(indexTmpl[0]);
		resp.write('/'+escape(board)+'/ - #' + op);
		resp.write(indexTmpl[1]);
		resp.write(make_thread_meta(board, num, limit));
		resp.write(indexTmpl[2]);
		resp.write('Thread #' + op);
		resp.write(indexTmpl[3]);
		resp.write('<hr>\n');
	});
	var opts = {fullPosts: true, ident: req.ident, board: board};
	write_thread_html(reader, resp, opts);
	reader.on('end', function () {
		resp.write(returnHTML);
		write_page_end(req, resp);
		yaku.disconnect();
	});
	function on_err(err) {
		winston.error('thread '+num+':', err);
		resp.end();
		yaku.disconnect();
	}
	reader.on('error', on_err);
	yaku.on('error', on_err);
	return true;
});
web.route_get(/^\/(\w+)\/(\d+)\/$/, function (req, resp, params) {
	var board = params[1];
	if (caps.under_curfew(req.ident, board))
		return web.redirect(resp, '..', 302);
	web.redirect(resp, '../' + params[2]);
});

function write_page_end(req, resp) {
	resp.write(RES.indexTmpl[4]);
	if (req.ident) {
		if (req.ident.auth == 'Admin')
			resp.write('<script src="../admin.js"></script>\n');
		else if (req.ident.auth == 'Moderator')
			resp.write('<script src="../mod.js"></script>\n');
	}
	resp.end();
}

web.route_get(/^\/outbound\/([\w+\/]{22})$/, function (req, resp, params) {
	// TEMP
	var service = 'http://archive.foolz.us/a/image/';
	var headers = {Location: service + escape(params[1]) + '/',
			'X-Robots-Tag': 'nofollow'};
	resp.writeHead(303, headers);
	resp.end();
});

web.route_get_auth(/^\/dead\/(src|thumb)\/(\w+\.\w{3})$/,
			function (req, resp, params) {
	if (req.ident.auth != 'Admin')
		return web.render_404(resp);
	pix.send_dead_image(params[1], params[2], resp);
});


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
	if (!this.synced && type != common.SYNCHRONIZE)
		type = common.INVALID;
	var func = dispatcher[type];
	if (!func || !func(msg, this)) {
		winston.warn("Got invalid message " + data);
		report(null, this, "Bad protocol.");
	}
};

OK.on_close = function () {
	if (this.id) {
		delete clients[this.id];
		this.id = null;
	}
	this.synced = false;
	var db = this.db;
	if (db) {
		db.kikanai();
		if (this.post)
			this.finish_post(function () {
				db.disconnect();
			});
		else
			db.disconnect();
	}
};

function pad3(n) {
	return (n < 10 ? '00' : (n < 100 ? '0' : '')) + n;
}

function report(error, client, client_msg) {
	var error_db = new db.Yakusoku(null, db.UPKEEP_IDENT);
	var msg = client_msg || 'Server error.';
	var ip = client && client.ident.ip;
	winston.error('Error by ' + ip + ': ' + (error || msg));
	if (client) {
		client.send([0, common.INVALID, msg]);
		client.synced = false;
	}
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
	callback(null, _.isEmpty(links) ? null : links);
}

dispatcher[common.INSERT_POST] = function (msg, client) {
	if (msg.length != 1)
		return false;
	msg = msg[0];
	if (!msg || typeof msg != 'object')
		return false;
	if (client.post)
		return update_post(msg.frag, client);
	var frag = msg.frag;
	if (frag && (typeof frag != 'string' || frag.match(/^\s*$/g)))
		return false;
	if (!frag && !msg.image)
		return false;
	if (config.DEBUG)
		debug_command(client, frag);

	allocate_post(msg, client, function (err, alloc) {
		if (err) {
			var niceErr = "Couldn't post: " + err;
			/* TEMP: Need better nice-error-message policy */
			if (niceErr.length > 40)
				niceErr = "Couldn't allocate post.";
			return report(err, client, niceErr);
		}
	});
	return true;
}

function allocate_post(msg, client, callback) {
	if (!msg || typeof msg != 'object')
		return callback('Bad alloc.');
	if (typeof msg.nonce != 'number' || !msg.nonce || msg.nonce < 1)
		return callback('Bad nonce.');
	if (client.post)
		return callback("Already have a post.");
	if (['graveyard', 'archive'].indexOf(client.board) >= 0)
		return callback("Can't post here.");
	var post = {time: new Date().getTime(), nonce: msg.nonce};
	var body = '';
	var ip = client.ident.ip;
	var extra = {ip: ip, board: client.board};
	var image_alloc;
	if (msg.image !== undefined) {
		if (typeof msg.image != 'string' || !msg.image.match(/^\d+$/))
			return callback('Bad image token.');
		image_alloc = msg.image;
	}
	if (msg.frag !== undefined) {
		if (typeof msg.frag != 'string' || msg.frag.match(/^\s*$/g))
			return callback('Bad post body.');
		if (msg.frag.length > common.MAX_POST_CHARS)
			return callback('Post is too long.');
		body = msg.frag.replace(config.EXCLUDE_REGEXP, '');
		if (config.GAME_BOARDS.indexOf(client.board) >= 0)
			amusement.roll_dice(body, post, extra);
	}
	if (msg.op !== undefined) {
		if (typeof msg.op != 'number' || msg.op < 1)
			return callback('Invalid thread.');
		post.op = msg.op;
	}
	else if (!image_alloc)
		return callback('Image missing.');
	/* TODO: Check against client.watching? */
	if (msg.name !== undefined) {
		if (typeof msg.name != 'string')
			return callback('Invalid name.');
		var parsed = common.parse_name(msg.name);
		post.name = parsed[0];
		var spec = STATE.hot.SPECIAL_TRIPCODES;
		if (spec && parsed[1] && parsed[1] in spec) {
			post.trip = spec[parsed[1]];
		}
		else if (parsed[1] || parsed[2]) {
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
	post.state = [common.S_BOL, 0];

	if (typeof msg.auth != 'undefined') {
		if (!client.ident || msg.auth !== client.ident.auth)
			return callback('Bad auth.');
		post.auth = msg.auth;
	}

	if (post.op)
		throttled(null);
	else
		client.db.check_throttle(ip, throttled);

	function throttled(err) {
		if (err)
			return callback(err);
		client.db.reserve_post(post.op, ip, got_reservation);
	}

	function got_reservation(err, num) {
		if (err)
			return callback(err);
		if (client.post)
			return callback('Already have a post.');
		client.post = post;
		post.num = num;
		var supplements = {
			links: valid_links.bind(null, body, post.state),
		};
		if (image_alloc)
			supplements.image = client.db.obtain_image_alloc.bind(
					client.db, image_alloc);
		async.parallel(supplements, got_supplements);
	}
	function got_supplements(err, rs) {
		if (err) {
			winston.error('supplements: ' + err);
			if (client.post === post)
				client.post = null;
			return callback("Attachment error.");
		}
		post.links = rs.links;
		if (rs.image)
			extra.image_alloc = rs.image;
		client.db.insert_post(post, body, extra, inserted);
	}
	function inserted(err) {
		if (err) {
			if (client.post === post)
				client.post = null;
			winston.error(err);
			return callback("Couldn't allocate post.");
		}
		post.body = body;
		callback(null, get_post_view(post));
	}
	return true;
}

function get_post_view(post) {
	var view = {num: post.num, body: post.body, time: post.time};
	if (post.nonce) view.nonce = post.nonce;
	if (post.op) view.op = post.op;
	if (post.name) view.name = post.name;
	if (post.trip) view.trip = post.trip;
	if (post.email) view.email = post.email;
	if (post.editing) view.editing = post.editing;
	if (post.links) view.links = post.links;
	if (post.image) view.image = post.image;
	if (post.dice) view.dice = post.dice;
	if (post.auth) view.auth = post.auth;
	return view;
}

function update_post(frag, client) {
	if (typeof frag != 'string')
		return false;
	if (config.DEBUG)
		debug_command(client, frag);
	frag = frag.replace(config.EXCLUDE_REGEXP, '');
	var post = client.post;
	if (!post)
		return false;
	var limit = common.MAX_POST_CHARS;
	if (frag.length > limit || post.length >= limit)
		return false;
	var combined = post.length + frag.length;
	if (combined > limit)
		frag = frag.substr(0, combined - limit);
	var extra = {ip: client.ident.ip};
	if (config.GAME_BOARDS.indexOf(client.board) >= 0)
		amusement.roll_dice(frag, post, extra);
	post.body += frag;
	/* imporant: broadcast prior state */
	var old_state = post.state.slice();

	valid_links(frag, post.state, function (err, links) {
		if (err)
			links = null; /* oh well */
		if (links) {
			if (!post.links)
				post.links = {};
			var new_links = {};
			for (var k in links) {
				var link = links[k];
				if (post.links[k] != link) {
					post.links[k] = link;
					new_links[k] = link;
				}
			}
			extra.links = links;
			extra.new_links = new_links;
		}

		client.db.append_post(post, frag, old_state, extra,
					function (err) {
			if (err)
				report(err, client, "Couldn't add text.");
		});
	});
	return true;
}
dispatcher[common.UPDATE_POST] = update_post;

function debug_command(client, frag) {
	if (!frag)
		return;
	if (frag.match(/\bfail\b/))
		report("debug", client, "Failure requested.");
	else if (frag.match(/\bclose\b/))
		client.socket.close();
}

OK.finish_post = function (callback) {
	/* TODO: Should we check this.uploading? */
	var self = this;
	this.db.finish_post(this.post, function (err) {
		if (err)
			callback(err);
		else {
			self.last_num = self.post.num;
			self.post = null;
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

dispatcher[common.DELETE_POSTS] = function (nums, client) {
	if (!caps.is_mod_ident(client.ident))
		return false;
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
};

dispatcher[common.DELETE_IMAGES] = function (nums, client) {
	if (!caps.is_mod_ident(client.ident))
		return false;
	if (!nums.length)
		return false;
	if (nums.some(function (n) { return typeof n != 'number' || n < 1; }))
		return false;

	client.db.remove_images(nums, function (err, dels) {
		if (err)
			report(err, client, "Couldn't delete images.");
	});
	return true;
};

dispatcher[common.INSERT_IMAGE] = function (msg, client) {
	if (msg.length != 1)
		return false;
	var alloc = msg[0];
	if (!alloc || typeof alloc != 'string')
		return false;
	if (!client.post || client.post.image)
		return false;
	client.db.obtain_image_alloc(alloc, function (err, alloc) {
		if (!client.post || client.post.image)
			return;
		client.db.add_image(client.post, alloc, client.ident.ip,
					function (err) {
			if (err)
				report(err, client, "Image insertion error.");
		});
	});
	return true;
};

dispatcher[common.SPOILER_IMAGES] = function (nums, client) {
	/* grr copy pasted */
	if (!caps.is_mod_ident(client.ident))
		return false;
	if (!nums.length)
		return false;
	if (nums.some(function (n) { return typeof n != 'number' || n < 1; }))
		return false;

	client.db.force_image_spoilers(nums, function (err) {
		if (err)
			report(err, client, "Couldn't spoiler images.");
	});
	return true;
};

dispatcher[common.EXECUTE_JS] = function (msg, client) {
	if (!caps.is_admin_ident(client.ident))
		return false;
	if (typeof msg[0] != 'number')
		return false;
	var op = msg[0];
	client.db.set_fun_thread(op, function (err) {
		if (err)
			report(err, client, "No fun allowed.");
	});
	return true;
};

function propagate_resources() {
	if (!tripcode.setSalt(config.SECURE_SALT))
		throw "Bad SECURE_SALT";
	web.notFoundHtml = RES.notFoundHtml;
}

function get_sockjs_script_sync() {
	var src = fs.readFileSync('tmpl/index.html', 'UTF-8');
	return src.match(/sockjs-[\d.]+(?:\.min)?\.js/)[0];
}

function sockjs_log(sev, message) {
	if (sev == 'info')
		winston.verbose(message);
	else if (sev == 'error')
		winston.error(message);
}
if (config.DEBUG) {
	winston.remove(winston.transports.Console);
	winston.add(winston.transports.Console, {level: 'verbose'});
}

function start_server() {
	web.server.listen(config.LISTEN_PORT, config.LISTEN_HOST);
	if (config.DEBUG)
		web.enable_debug();
	var sockjsPath = 'js/' + get_sockjs_script_sync();
	var sockOpts = {
		sockjs_url: config.MEDIA_URL + sockjsPath,
		prefix: config.SOCKET_PATH,
		jsessionid: false,
		log: sockjs_log,
	};
	var sockJs = require('sockjs').createServer(sockOpts);
	web.server.on('upgrade', function (req, resp) {
		resp.end();
	});
	sockJs.installHandlers(web.server);

	sockJs.on('connection', function (socket) {
		var ip = socket.remoteAddress;
		if (config.TRUST_X_FORWARDED_FOR) {
			var ff = web.parse_forwarded_for(
					socket.headers['x-forwarded-for']);
			if (ff)
				ip = ff;
		}

		var client = new Okyaku(socket, ip);
		socket.on('data', client.on_message.bind(client));
		socket.on('close', client.on_close.bind(client));
	});

	process.on('SIGHUP', function () {
		async.series([
			STATE.reload_hot,
			STATE.reset_resources,
		], function (err) {
			if (err)
				throw err;
			propagate_resources();
			winston.info('Reloaded initial state.');
		});
	});

	if (config.DAEMON) {
		var cfg = config.DAEMON;
		var daemon = require('daemon');
		var pid = daemon.start(process.stdout.fd, process.stderr.fd);
		var lock = require('path').join(cfg.PID_PATH, 'server.pid');
		daemon.lock(lock);
		winston.remove(winston.transports.Console);
	}
}

if (require.main == module) {
	async.series([
		STATE.reload_hot,
		STATE.make_media_dirs,
		STATE.reset_resources,
		db.track_OPs,
	], function (err) {
		if (err)
			throw err;
		propagate_resources();
		var yaku = new db.Yakusoku(null, db.UPKEEP_IDENT);
		async.series([
			yaku.finish_all.bind(yaku),
			yaku.delete_temporaries.bind(yaku),
		], function (err) {
			if (err)
				throw err;
			yaku.disconnect();
			_.defer(start_server);
		});
	});
}
