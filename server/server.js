var opts = require('./opts');
if (require.main == module) opts.parse_args();
opts.load_defaults();

var _ = require('../lib/underscore'),
    amusement = require('./amusement'),
    async = require('async'),
    caps = require('./caps'),
    check = require('./msgcheck').check,
    common = require('../common'),
    config = require('../config'),
    db = require('../db'),
    fs = require('fs'),
    hooks = require('../hooks'),
    imager = require('../imager'),
    Muggle = require('../etc').Muggle,
    okyaku = require('./okyaku'),
    persona = require('./persona'),
    render = require('./render'),
    STATE = require('./state'),
    tripcode = require('./../tripcode/tripcode'),
    urlParse = require('url').parse,
    web = require('./web'),
    winston = require('winston');

require('../admin');
if (!imager.is_standalone())
	require('../imager/daemon'); // preload and confirm it works
if (config.CURFEW_BOARDS)
	require('../curfew/server');
var anon_hours;
if (config.ANON_HOURS)
	anon_hours = require('../anon_hours/server');
var autoJoe;
if (config.AUTOJOE){
	autoJoe = require('../autoJoe/server');
	autoJoe.isJoe = false;
}
try {
	var reportConfig = require('../report/config');
	if (reportConfig.RECAPTCHA_PUBLIC_KEY)
		require('../report/server');
} catch (e) {}
require('../time/server');
if (config.VOICE_PATH)
	require('../voice/server');

var RES = STATE.resources;

var dispatcher = okyaku.dispatcher;

/* I always use encodeURI anyway */
var escape = common.escape_html;
var safe = common.safe;

dispatcher[common.SYNCHRONIZE] = function (msg, client) {
	function checked(err, ident) {
		if (!err)
			_.extend(client.ident, ident);
		if (!synchronize(msg, client))
			client.kotowaru(Muggle("Bad protocol."));
	}
	var chunks = web.parse_cookie(msg.pop());
	var cookie = persona.extract_login_cookie(chunks);
	if (cookie) {
		persona.check_cookie(cookie, checked);
		return true;
	}
	else
		return synchronize(msg, client);
};

function synchronize(msg, client) {
	if (!check(['id', 'string', 'id=>nat', 'boolean'], msg))
		return false;
	var id = msg[0], board = msg[1], syncs = msg[2], live = msg[3];
	if (id in STATE.clients) {
		winston.error("Duplicate client id " + id);
		return false;
	}
	client.id = id;
	STATE.clients[id] = client;
	if (client.synced) {
		//winston.warn("Client tried to sync twice");
		/* Sync logic is buggy; allow for now */
		//return true;
	}
	if (!caps.can_access_board(client.ident, board))
		return false;
	var dead_threads = [], count = 0, op;
	for (var k in syncs) {
		k = parseInt(k, 10);
		if (db.OPs[k] != k || !db.OP_has_tag(board, k)) {
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
			return client.kotowaru(Muggle(
					"Couldn't sync to board."));
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

		var info = {client: client, live: live};
		if (!live && count == 1)
			info.op = op;
		else
			info.board = board;
		hooks.trigger('clientSynced', info, function (err) {
			if (err)
				winston.error(err);
		});
	}
	return true;
}

function setup_imager_relay(cb) {
	var onegai = new imager.Onegai;
	onegai.relay_client_messages();
	onegai.once('relaying', function () {
		onegai.on('message', image_status);
		cb(null);
	});
}

function image_status(client_id, status) {
	if (!check('id', client_id))
		return;
	var client = STATE.clients[client_id];
	if (client) {
		try {
			client.send([0, common.IMAGE_STATUS, status]);
		}
		catch (e) {
			// Swallow EINTR
			// anta baka?
		}
	}
}

function page_nav(thread_count, cur_page, ascending) {
	var page_count = Math.ceil(thread_count / config.THREADS_PER_PAGE);
	page_count = Math.max(page_count, 1);
	var info = {pages: page_count, threads: thread_count,
		cur_page: cur_page, ascending: ascending};

	var step = ascending ? -1 : 1;
	var next = Math.max(cur_page, 0) + step;
	if (next >= 0 && next < page_count)
		info.next_page = 'page' + next;
	var prev = cur_page - step;
	if (prev >= 0 && prev < page_count)
		info.prev_page = 'page' + prev;
	return info;
}

function write_gzip_head(req, resp, headers) {
	var encoding = config.GZIP && req.headers['accept-encoding'];
	if (req.ident.slow || !encoding || encoding.indexOf('gzip') < 0) {
		resp.writeHead(200, headers);
		return resp;
	}
	resp.writeHead(200, _.extend({}, headers, {
		'Content-Encoding': 'gzip',
		Vary: 'Accept-Encoding',
	}));

	var gz = require('zlib').createGzip();
	gz.pipe(resp);
	return gz;
}

function redirect_thread(cb, num, op, tag) {
	if (!tag)
		cb(null, 'redirect', op + '#' + num);
	else
		/* Use a JS redirect to preserve the hash */
		cb(null, 'redirect_js', '../' + tag + '/' + op + '#' + num);
}

// unless imager.config.DAEMON, we deal with image uploads in-process.
if (!imager.is_standalone()) {
	web.route_post(/^\/upload\/$/, require('../imager/daemon').new_upload);
}

web.resource(/^\/$/, function (req, cb) {
	cb(null, 'redirect', config.DEFAULT_BOARD + '/');
});

web.route_post(/^\/login$/, persona.login);
web.route_post_auth(/^\/logout$/, persona.logout);
if (config.DEBUG) {
	/* Shortcuts for convenience */
	winston.warn("Running in (insecure) debug mode.");
	winston.warn("Do not use on the public internet.");
	web.route_get(/^\/login$/, function (req, resp) {
		persona.set_cookie(resp, {auth: 'Admin'});
	});
	web.route_get(/^\/mod$/, function (req, resp) {
		persona.set_cookie(resp, {auth: 'Moderator'});
	});
	web.route_get(/^\/logout$/, persona.logout);
}
else {
	/* Production login/out endpoint */
	web.resource(/^\/login$/, true, function (req, resp) {
		resp.writeHead(200, web.noCacheHeaders);
		resp.write(RES.loginTmpl[0]);
		resp.write('{}');
		resp.end(RES.loginTmpl[1]);
	});

	web.resource(/^\/logout$/, function (req, cb) {
		if (req.ident.auth)
			cb(null, 'ok');
		else
			cb(null, 'redirect', config.DEFAULT_BOARD+'/');
	},
	function (req, resp) {
		resp.writeHead(200, web.noCacheHeaders);
		resp.write(RES.loginTmpl[0]);
		resp.write(JSON.stringify({
			loggedInUser: req.ident.email,
			x_csrf: req.ident.csrf,
		}));
		resp.end(RES.loginTmpl[1]);
	});
}
web.resource(/^\/(login|logout)\/$/, function (req, params, cb) {
	cb(null, 'redirect', '../' + params[1]);
});

function write_mod_js(resp, ident) {
	if (!RES.modJs) {
		resp.writeHead(500);
		resp.end('Mod js not built?!');
		return;
	}

	var noCacheJs = _.clone(web.noCacheHeaders);
	noCacheJs['Content-Type'] = 'text/javascript; charset=UTF-8';
	resp.writeHead(200, noCacheJs);
	resp.write('(function (IDENT) {');
	resp.write(RES.modJs);
	resp.end('})(' + JSON.stringify(ident) + ');');
}

web.resource_auth(/^\/admin\.js$/, function (req, cb) {
	if (!caps.can_administrate(req.ident))
		cb(404);
	else
		cb(null, 'ok');
},
function (req, resp) {
	write_mod_js(resp, {
		auth: req.ident.auth,
		csrf: req.ident.csrf,
		email: req.ident.email,
	});
});

web.resource_auth(/^\/mod\.js$/, function (req, cb) {
	if (!caps.can_moderate(req.ident))
		cb(404);
	else
		cb(null, 'ok');
},
function (req, resp) {
	write_mod_js(resp, {
		auth: req.ident.auth,
		csrf: req.ident.csrf,
		email: req.ident.email,
	});
});

web.resource(/^\/(\w+)$/, function (req, params, cb) {
	var board = params[1];
	/* If arbitrary boards were allowed, need to escape this: */
	var dest = board + '/';
	if (req.ident.suspension)
		return cb(null, 'redirect', dest); /* TEMP */
	if (!caps.can_ever_access_board(req.ident, board))
		return cb(404);
	cb(null, 'redirect', dest);
});

web.resource(/^\/(\w+)\/live$/, function (req, params, cb) {
	if (req.ident.suspension)
		return cb(null, 'redirect', '.'); /* TEMP */
	if (!caps.can_ever_access_board(req.ident, params[1]))
		return cb(404);
	cb(null, 'redirect', '.');
});

web.resource(/^\/(\w+)\/$/, function (req, params, cb) {
	var board = params[1];
	if (req.ident.suspension)
		return cb(null, 'ok'); /* TEMP */
	if (!caps.can_ever_access_board(req.ident, board))
		return cb(404);

	// we don't do board etags yet
	var info = {etag: 'dummy', req: req};
	hooks.trigger_sync('buildETag', info);

	cb(null, 'ok', {board: board});
},
function (req, resp) {
	/* TEMP */
	if (req.ident.suspension)
		return render_suspension(req, resp);

	var board = this.board;
	// Only render <threads> for pushState() updates
	var min = !!req.query.minimal;
	var info = {board: board, ident: req.ident, resp: resp};
	hooks.trigger_sync('boardDiversion', info);
	if (info.diverted)
		return;

	var yaku = new db.Yakusoku(board, req.ident);
	yaku.get_tag(-1);
	var paginationHtml;
	yaku.once('begin', function (thread_count) {
		var nav = page_nav(thread_count, -1, board == 'archive');
		if (!min)
			render.write_board_head(resp, board, nav);
		else
			render.write_board_title(resp, board);
		paginationHtml = render.make_pagination_html(nav);
		resp.write(paginationHtml);
		resp.write('<hr class="sectionHr">\n');
	});
	resp = write_gzip_head(req, resp, web.noCacheHeaders);
	var opts = {fullLinks: true, board: board};
	render.write_thread_html(yaku, req, resp, opts);
	yaku.once('end', function () {
		resp.write(paginationHtml);
		render.write_page_end(resp, req.ident, false, min);
		resp.end();
		yaku.disconnect();
	});
	yaku.once('error', function (err) {
		winston.error('index:' + err);
		resp.end();
		yaku.disconnect();
	});
});

web.resource(/^\/(\w+)\/page(\d+)$/, function (req, params, cb) {
	var board = params[1];
	if (!caps.temporal_access_check(req.ident, board))
		return cb(null, 302, '..');
	if (req.ident.suspension)
		return cb(null, 'ok'); /* TEMP */
	if (!caps.can_access_board(req.ident, board))
		return cb(404);
	var page = parseInt(params[2], 10);
	if (page > 0 && params[2][0] == '0') /* leading zeroes? */
		return cb(null, 'redirect', 'page' + page);

	var yaku = new db.Yakusoku(board, req.ident);
	yaku.get_tag(page);
	yaku.once('nomatch', function () {
		cb(null, 302, '.');
		yaku.disconnect();
	});
	yaku.once('begin', function (threadCount) {
		// we don't do board etags yet
		var info = {etag: 'dummy', req: req};
		hooks.trigger_sync('buildETag', info);

		cb(null, 'ok', {
			board: board, page: page, yaku: yaku,
			threadCount: threadCount,
		});
	});
},
function (req, resp) {
	/* TEMP */
	if (req.ident.suspension)
		return render_suspension(req, resp);

	var board = this.board;
	var min = !!req.query.minimal;
	console.log(min);
	var nav = page_nav(this.threadCount, this.page, board == 'archive');
	resp = write_gzip_head(req, resp, web.noCacheHeaders);
	if (!min)
		render.write_board_head(resp, board, nav);
	else
		render.write_board_title(resp, board);
	var paginationHtml = render.make_pagination_html(nav);
	resp.write(paginationHtml);
	resp.write('<hr class="sectionHr">\n');

	var opts = {fullLinks: true, board: board};
	render.write_thread_html(this.yaku, req, resp, opts);
	var self = this;
	this.yaku.once('end', function () {
		resp.write(paginationHtml);
		render.write_page_end(resp, req.ident, false, min);
		resp.end();
		self.finished();
	});
	this.yaku.once('error', function (err) {
		winston.error('page' + self.page + ': ' + err);
		resp.end();
		self.finished();
	});
},
function () {
	this.yaku.disconnect();
});

web.resource(/^\/(\w+)\/page(\d+)\/$/, function (req, params, cb) {
	if (!caps.temporal_access_check(req.ident, params[1]))
		cb(null, 302, '..');
	else
		cb(null, 'redirect', '../page' + params[2]);
});

web.resource(/^\/(\w+)\/(\d+)$/, function (req, params, cb) {
	var board = params[1];
	if (!caps.temporal_access_check(req.ident, board))
		return cb(null, 302, '.');
	if (req.ident.suspension)
		return cb(null, 'ok'); /* TEMP */
	if (!caps.can_access_board(req.ident, board))
		return cb(404);
	var num = parseInt(params[2], 10);
	if (!num)
		return cb(404);
	else if (params[2][0] == '0')
		return cb(null, 'redirect', '' + num);

	var op, json = web.prefers_json(req.headers.accept);
	if (board == 'graveyard') {
		op = num;
	}
	else {
		op = db.OPs[num];
		if (!op)
			return cb(404);
		if (!json && !db.OP_has_tag(board, op)) {
			var tag = db.first_tag_of(op);
			if (tag) {
				if (!caps.can_access_board(req.ident, tag))
					return cb(404);
				return redirect_thread(cb, num, op, tag);
			}
			else {
				winston.warn("Orphaned post " + num +
					"with tagless OP " + op);
				return cb(404);
			}
		}
		if (!json && op != num)
			return redirect_thread(cb, num, op);
	}
	if (!caps.can_access_thread(req.ident, op))
		return cb(404);
	if (json)
		return cb(null, 'ok', {json: true, num: num});

	var yaku = new db.Yakusoku(board, req.ident);
	var reader = new db.Reader(yaku);
	var opts = {redirect: true};

	var lastN = detect_last_n(req.query);
	if (lastN)
		opts.abbrev = lastN + config.ABBREVIATED_REPLIES;

	if (caps.can_administrate(req.ident) && 'showdead' in req.query)
		opts.showDead = true;
	reader.get_thread(board, num, opts);
	reader.once('nomatch', function () {
		cb(404);
		yaku.disconnect();
	});
	reader.once('redirect', function (op) {
		redirect_thread(cb, num, op);
		yaku.disconnect();
	});
	reader.once('begin', function (preThread) {
		var headers;
		if (!config.DEBUG && preThread.hctr) {
			var etag = 'W/' + preThread.hctr + '-' + RES.indexHash;
			var chunks = web.parse_cookie(req.headers.cookie);
			var thumb = req.cookies.thumb;
			if (thumb && common.thumbStyles.indexOf(thumb) >= 0)
				etag += '-' + thumb;
			if (chunks.spoil == 'true' || chunks.spoil == 'false')
				etag += '-sp_' + chunks.spoil;
			if (chunks.agif == 'true' || chunks.agif == 'false')
				etag += '-ag_' + chunks.agif;
			if (lastN)
				etag += '-last' + lastN;
			if (preThread.locked)
				etag += '-locked';
			if (req.ident.auth)
				etag += '-auth';

			var info = {etag: etag, req: req};
			hooks.trigger_sync('buildETag', info);
			etag = info.etag;

			if (req.headers['if-none-match'] === etag) {
				yaku.disconnect();
				return cb(null, 304);
			}
			headers = _.clone(web.vanillaHeaders);
			headers.ETag = etag;
			headers['Cache-Control'] = (
					'private, max-age=0, must-revalidate');
		}
		else
			headers = web.noCacheHeaders;

		cb(null, 'ok', {
			headers: headers,
			board: board, op: op,
			subject: preThread.subject,
			yaku: yaku, reader: reader,
			abbrev: opts.abbrev,
		});
	});
},
function (req, resp) {
	/* TEMP */
	if (req.ident.suspension)
		return render_suspension(req, resp);
	if (this.json)
		return write_json_post(req, resp, this.num);

	var board = this.board, op = this.op;
	var min = !!req.query.minimal;

	resp = write_gzip_head(req, resp, this.headers);
	if (!min){
		render.write_thread_head(resp, board, op, {
			subject: this.subject,
			abbrev: this.abbrev,
		});
	}
	else {
		render.write_thread_title(resp, board, op, {
			subject: this.subject,
			abbrev: this.abbrev,
		});
	}
	var opts = {fullPosts: true, board: board, loadAllPostsLink: true};
	render.write_thread_html(this.reader, req, resp, opts);
	var self = this;
	this.reader.once('end', function () {
		render.write_page_end(resp, req.ident, true, min);
		resp.end();
		self.finished();
	});
	function on_err(err) {
		winston.error('thread '+num+':', err);
		resp.end();
		self.finished();
	}
	this.reader.once('error', on_err);
	this.yaku.once('error', on_err);
},
function () {
	this.yaku.disconnect();
});

function write_json_post(req, resp, num) {
	var json = {TODO: true};

	var cache = json.editing ? 'no-cache' : 'private, max-age=86400';
	resp = write_gzip_head(req, resp, {
		'Content-Type': 'application/json',
		'Cache-Control': cache,
	});
	resp.end(JSON.stringify(json));
}

function detect_last_n(query) {
	if (!!query.last){
		var n = parseInt(query.last);
		if (common.reasonable_last_n(n))
			return n;
	}
	return 0;
}

web.resource(/^\/(\w+)\/(\d+)\/$/, function (req, params, cb) {
	if (!caps.temporal_access_check(req.ident, params[1]))
		cb(null, 302, '..');
	else
		cb(null, 'redirect', '../' + params[2]);
});

web.resource(/^\/outbound\/(g|iqdb|sn)\/(\d+\.jpg)$/,
			function (req, params, cb) {
	var thumb = imager.config.MEDIA_URL + 'thumb/' + params[2];

	// attempt to make protocol more absolute
	var u = urlParse(thumb, false, true);
	if (!u.protocol) {
		u.protocol = 'http:';
		thumb = u.format();
	}

	// Pass unencrypted URL to IQDB and SauceNao to avoid problems with Cloudflare's SSL
	if ((params[1] == 'iqdb' || params[1] == 'sn') && imager.config.NO_SSL_QUERY_STRING)
		thumb = thumb.replace(/https:\/\//, 'http://') + imager.config.NO_SSL_QUERY_STRING;

	if (params[1] == 'iqdb')
		var service = 'http://iqdb.org/?url=';
	else if (params[1] == 'g')
		var service = 'https://www.google.com/searchbyimage?image_url=';
	else
		var service = 'http://saucenao.com/search.php?db=999&url=';
	
	var dest = service + encodeURIComponent(thumb);
	cb(null, 303.1, dest);
});

web.resource(/^\/outbound\/(hash|exh)\/([\w+\/]{22}|[\w+\/]{40})$/, function (req, params, cb) {
	var url = params[1] == 'hash' ? 'http://archive.foolz.us/_/search/image/' :
		'http://exhentai.org/?fs_similar=1&fs_exp=1&f_shash=';
	var dest = url + escape(params[2]);
	cb(null, 303.1, dest);
});

web.resource(/^\/outbound\/a\/(\d{0,10})$/, function (req, params, cb) {
	var thread = parseInt(params[1], 10);
	if (thread)
		cb(null, 'ok');
	else
		cb(null, 303.1, 'http://boards.4chan.org/a/');
}, function (req, resp) {
	resp.writeHead(200, web.noCacheHeaders);
	resp.end(RES.aLookupHtml);
});

web.resource(/^\/outbound\/foolz\/(\d{0,10})$/, function (req, params, cb) {
	var dest = 'http://archive.foolz.us/foolz/';
	var thread = parseInt(params[1], 10);
	cb(null, 303.1, thread ? dest+'thread/'+thread+'/' : dest);
});

web.route_get_auth(/^\/dead\/(src|thumb|mid)\/(\w+\.\w{3})$/,
			function (req, resp, params) {
	if (!caps.can_administrate(req.ident))
		return web.render_404(resp);
	imager.send_dead_image(params[1], params[2], resp);
});


/* Must be prepared to receive callback instantly */
function valid_links(frag, state, ident, callback) {
	var links = {};
	var onee = new common.OneeSama(function (num) {
		var op = db.OPs[num];
		if (op && caps.can_access_thread(ident, op))
			links[num] = db.OPs[num];
	});
	onee.callback = function (frag) {};
	onee.state = state;
	onee.fragment(frag);
	callback(null, _.isEmpty(links) ? null : links);
}

var insertSpec = [{
	frag: 'opt string',
	image: 'opt string',
	nonce: 'id',
	op: 'opt id',
	name: 'opt string',
	email: 'opt string',
	auth: 'opt string',
	subject: 'opt string',
}];

dispatcher[common.INSERT_POST] = function (msg, client) {
	if (!check(insertSpec, msg))
		return false;
	msg = msg[0];
	if (client.post)
		return update_post(msg.frag, client);
	if (!caps.can_access_board(client.ident, client.board))
		return false;
	var frag = msg.frag;
	if (frag && /^\s*$/g.test(frag))
		return false;
	if (!frag && !msg.image)
		return false;
	if (config.DEBUG)
		debug_command(client, frag);

	allocate_post(msg, client, function (err) {
		if (err)
			client.kotowaru(Muggle("Allocation failure.", err));
	});
	return true;
}

function inactive_board_check(client) {
	if (caps.can_administrate(client.ident))
		return true;
	return ['graveyard', 'archive'].indexOf(client.board) == -1;
}

function allocate_post(msg, client, callback) {
	if (client.post)
		return callback(Muggle("Already have a post."));
	if (!inactive_board_check(client))
		return callback(Muggle("Can't post here."));
	var post = {time: Date.now(), nonce: msg.nonce};
	var body = '';
	var ip = client.ident.ip;
	var extra = {ip: ip, board: client.board};
	var image_alloc;
	if (msg.image) {
		if (!/^\d+$/.test(msg.image))
			return callback(Muggle('Expired image token.'));
		image_alloc = msg.image;
	}
	if (msg.frag) {
		if (/^\s*$/g.test(msg.frag))
			return callback(Muggle('Bad post body.'));
		if (msg.frag.length > common.MAX_POST_CHARS)
			return callback(Muggle('Post is too long.'));
		body = hot_filter(msg.frag.replace(config.EXCLUDE_REGEXP, ''));
	}

	if (msg.op) {
		if (db.OPs[msg.op] != msg.op)
			return callback(Muggle('Thread does not exist.'));
		if (!db.OP_has_tag(extra.board, msg.op))
			return callback(Muggle('Thread does not exist.'));
		post.op = msg.op;
	}
	else {
		if (!image_alloc)
			return callback(Muggle('Image missing.'));
		var subject = (msg.subject || '').trim();
		subject = subject.replace(config.EXCLUDE_REGEXP, '');
		subject = subject.replace(/[「」]/g, '');
		subject = subject.slice(0, config.SUBJECT_MAX_LENGTH);
		if (subject)
			post.subject = subject;
	}

	// Anon hours name replacement
	if(anon_hours && (!autoJoe || !autoJoe.isJoe)){
		// Add name to used name list for random name hours
		if (msg.name)
			anon_hours.name_parse(msg.name);
		// Pull random name from name list
		if (anon_hours.random_name_hour)
			anon_hours.random_name(post);
	}
	// Replace names, when a song plays on r/a/dio
	if (autoJoe){
		if (autoJoe.isJoe)
			post.name = autoJoe.name;
	}
	/* TODO: Check against client.watching? */
	if (msg.name && (!anon_hours || !anon_hours.anon_hour) && (!autoJoe || !autoJoe.isJoe)) {
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
	if (msg.email) {
		post.email = msg.email.trim().substr(0, 320);
		if (common.is_noko(post.email))
			delete post.email;
	}
	post.state = [common.S_BOL, 0];

	if ('auth' in msg) {
		if (!msg.auth || !client.ident
				|| msg.auth !== client.ident.auth)
			return callback(Muggle('Bad auth.'));
		post.auth = msg.auth;
	}

	if (post.op)
		client.db.check_thread_locked(post.op, checked);
	else
		client.db.check_throttle(ip, checked);

	function checked(err) {
		if (err)
			return callback(err);
		client.db.reserve_post(post.op, ip, got_reservation);
	}

	function got_reservation(err, num) {
		if (err)
			return callback(err);
		if (!client.synced)
			return callback(Muggle('Dropped; post aborted.'));
		if (client.post)
			return callback(Muggle('Already have a post.'));

		if (body.length)
			amusement.roll_dice(body, post, extra);

		client.post = post;
		post.num = num;
		var supplements = {
			links: valid_links.bind(null, body, post.state,
					client.ident),
		};
		if (image_alloc)
			supplements.image = imager.obtain_image_alloc.bind(
					null, image_alloc);
		async.parallel(supplements, got_supplements);
	}
	function got_supplements(err, rs) {
		if (err) {
			if (client.post === post)
				client.post = null;
			return callback(Muggle("Attachment error.", err));
		}
		if (!client.synced)
			return callback(Muggle('Dropped; post aborted.'));
		post.links = rs.links;
		if (rs.image)
			extra.image_alloc = rs.image;
		client.db.insert_post(post, body, extra, inserted);
	}
	function inserted(err) {
		if (err) {
			if (client.post === post)
				client.post = null;
			return callback(Muggle("Couldn't allocate post.",err));
		}
		post.body = body;
		callback(null);
	}
	return true;
}

function update_post(frag, client) {
	if (typeof frag != 'string')
		return false;
	if (config.DEBUG)
		debug_command(client, frag);
	frag = hot_filter(frag.replace(config.EXCLUDE_REGEXP, ''));
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
	amusement.roll_dice(frag, post, extra);
	post.body += frag;
	/* imporant: broadcast prior state */
	var old_state = post.state.slice();

	valid_links(frag, post.state, client.ident, function (err, links) {
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
				client.kotowaru(Muggle("Couldn't add text.",
						err));
		});
	});
	return true;
}
dispatcher[common.UPDATE_POST] = update_post;

function debug_command(client, frag) {
	if (!frag)
		return;
	if (/\bfail\b/.test(frag))
		client.kotowaru(Muggle("Failure requested."));
	else if (/\bclose\b/.test(frag))
		client.socket.close();
}

dispatcher[common.FINISH_POST] = function (msg, client) {
	if (!check([], msg))
		return false;
	if (!client.post)
		return true; /* whatever */
	client.finish_post(function (err) {
		if (err)
			client.kotowaru(Muggle("Couldn't finish post.", err));
	});
	return true;
}

dispatcher[common.DELETE_POSTS] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.kotowaru(Muggle("Couldn't delete."));
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
			client.kotowaru(Muggle("Couldn't delete.", err));
	});
});

dispatcher[common.LOCK_THREAD] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.kotowaru(Muggle("Couldn't (un)lock thread."));
	nums = nums.filter(function (op) { return db.OPs[op] == op; });
	async.forEach(nums, client.db.toggle_thread_lock.bind(client.db),
				function (err) {
		if (err)
			client.kotowaru(Muggle(
					"Couldn't (un)lock thread.", err));
	});
});

dispatcher[common.DELETE_IMAGES] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.kotowaru(Muggle("Couldn't delete images."));
	client.db.remove_images(nums, function (err, dels) {
		if (err)
			client.kotowaru(Muggle("Couldn't delete images.",err));
	});
});

dispatcher[common.INSERT_IMAGE] = function (msg, client) {
	if (!check(['string'], msg))
		return false;
	var alloc = msg[0];
	if (!client.post || client.post.image)
		return false;
	imager.obtain_image_alloc(alloc, function (err, alloc) {
		if (err)
			return client.kotowaru(Muggle("Image lost.", err));
		if (!client.post || client.post.image)
			return;
		client.db.add_image(client.post, alloc, client.ident.ip,
					function (err) {
			if (err)
				client.kotowaru(Muggle(
					"Image insertion problem.", err));
		});
	});
	return true;
};

dispatcher[common.SPOILER_IMAGES] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.kotowaru(Muggle("Couldn't spoiler images."));
	client.db.force_image_spoilers(nums, function (err) {
		if (err)
			client.kotowaru(Muggle("Couldn't spoiler images.",
					err));
	});
});

dispatcher[common.EXECUTE_JS] = function (msg, client) {
	if (!caps.can_administrate(client.ident))
		return false;
	if (!check(['id'], msg))
		return false;
	var op = msg[0];
	client.db.set_fun_thread(op, function (err) {
		if (err)
			client.kotowaru(err);
	});
	return true;
};

// Regex replacement filter
function hot_filter(frag){
	var filter = STATE.hot.FILTER;
	if (!filter)
		return frag;
	for (i =0; i < filter.length; i++){
		var f = filter[i];
		var m = frag.match(f.p);
		if (m){
			// Case sensitivity
			if (m[0].length > 2){
				var first = m[0].charAt(0);
				var second = m[0].charAt(1);
				if (/[A-Z]/.test(second))
					f.r = f.r.toUpperCase();
				else if (/[A-Z]/.test(first)){
					console.log(f.r);
					f.r = f.r.charAt(0).toUpperCase()+f.r.slice(1);
				}
			}
			return frag.replace(f.p, f.r);
		}
	}
	return frag;
}

function render_suspension(req, resp) {
setTimeout(function () {
	var ban = req.ident.suspension, tmpl = RES.suspensionTmpl;
	resp.writeHead(200, web.noCacheHeaders);
	resp.write(tmpl[0]);
	resp.write(escape(ban.why || ''));
	resp.write(tmpl[1]);
	resp.write(escape(ban.until || ''));
	resp.write(tmpl[2]);
	resp.write(escape(STATE.hot.EMAIL || '<missing>'));
	resp.end(tmpl[3]);
}, 2000);
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
	var is_unix_socket = (typeof config.LISTEN_PORT == 'string');
	if (is_unix_socket) {
		try { fs.unlinkSync(config.LISTEN_PORT); } catch (e) {}
	}
	web.server.listen(config.LISTEN_PORT, config.LISTEN_HOST);
	if (is_unix_socket) {
		fs.chmodSync(config.LISTEN_PORT, '777'); // TEMP
	}


	var sockjsPath = 'js/' + get_sockjs_script_sync();
	var sockOpts = {
		sockjs_url: imager.config.MEDIA_URL + sockjsPath,
		prefix: config.SOCKET_PATH,
		jsessionid: false,
		log: sockjs_log,
		websocket: config.USE_WEBSOCKETS,
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

		var client = new okyaku.Okyaku(socket, ip);
		socket.on('data', client.on_message.bind(client));
		socket.on('close', client.on_close.bind(client));
	});

	process.on('SIGHUP', hot_reloader);
	db.on_pub('reloadHot', hot_reloader);

	if (config.DAEMON) {
		var cfg = config.DAEMON;
		var daemon = require('daemon');
		var pid = daemon.start(process.stdout.fd, process.stderr.fd);
		var lock = config.PID_FILE;
		daemon.lock(lock);
		winston.remove(winston.transports.Console);
		return;
	}

	process.nextTick(non_daemon_pid_setup);

	winston.info('Listening on '
			+ (config.LISTEN_HOST || '')
			+ (is_unix_socket ? '' : ':')
			+ (config.LISTEN_PORT + '.'));
}

function hot_reloader() {
	STATE.reload_hot_resources(function (err) {
		if (err) {
			winston.error("Error trying to reload:");
			winston.error(err);
			return;
		}
		okyaku.scan_client_caps();
		winston.info('Reloaded initial state.');
	});
}

function non_daemon_pid_setup() {
	var pidFile = config.PID_FILE;
	fs.writeFile(pidFile, process.pid+'\n', function (err) {
		if (err)
			return winston.warn("Couldn't write pid: " + err);
		process.once('SIGINT', delete_pid);
		process.once('SIGTERM', delete_pid);
		winston.info('PID ' + process.pid + ' written in ' + pidFile);
	});

	function delete_pid() {
		try {
			fs.unlinkSync(pidFile);
		}
		catch (e) { }
		process.exit(1);
	}
}

if (require.main == module) {
	if (!process.getuid())
		throw new Error("Refusing to run as root.");
	if (!tripcode.setSalt(config.SECURE_SALT))
		throw "Bad SECURE_SALT";
	async.series([
		imager.make_media_dirs,
		setup_imager_relay,
		STATE.reload_hot_resources,
		db.track_OPs,
	], function (err) {
		if (err)
			throw err;

		var yaku = new db.Yakusoku(null, db.UPKEEP_IDENT);
		var onegai;
		var writes = [];
		if (!config.READ_ONLY) {
			writes.push(yaku.finish_all.bind(yaku));
			if (!imager.is_standalone()) {
				onegai = new imager.Onegai;
				writes.push(onegai.delete_temporaries.bind(
						onegai));
			}
		}
		async.series(writes, function (err) {
			if (err)
				throw err;
			yaku.disconnect();
			if (onegai)
				onegai.disconnect();
			process.nextTick(start_server);
		});
	});
}
