var _ = require('../lib/underscore'),
    amusement = require('./amusement'),
    async = require('async'),
    caps = require('./caps'),
    check = require('./msgcheck').check,
    common = require('../common'),
    config = require('../config'),
    db = require('../db'),
    fs = require('fs'),
    get_version = require('../get').get_version,
    hooks = require('../hooks'),
    imager = require('../imager'),
    Muggle = require('../muggle').Muggle,
    okyaku = require('./okyaku'),
    persona = require('./persona'),
    STATE = require('./state'),
    tripcode = require('./../tripcode/tripcode'),
    web = require('./web'),
    winston = require('winston');

require('./panel');
require('../imager/daemon');
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
			client.report(Muggle("Bad protocol."));
	}
	var chunks = web.parse_cookie(msg.pop());
	cookie = persona.extract_login_cookie(chunks);
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
			return client.report(Muggle(
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

function tamashii(num) {
	var op = db.OPs[num];
	if (op && caps.can_access_thread(this.ident, op))
		this.callback(this.post_ref(num, op));
	else
		this.callback('>>' + num);
}

function write_thread_html(reader, req, response, opts) {
	var oneeSama = new common.OneeSama(tamashii);

	opts.ident = req.ident;
	caps.augment_oneesama(oneeSama, opts);
	var cookies = web.parse_cookie(req.headers.cookie);

	if (common.thumbStyles.indexOf(cookies.thumb) >= 0)
		oneeSama.thumbStyle = cookies.thumb;

	/* TEMP migration */
	if (cookies.img == 'no')
		oneeSama.thumbStyle = 'hide';

	reader.on('thread', function (op_post, omit, image_omit) {
		op_post.omit = omit;
		var full = oneeSama.full = !!opts.fullPosts;
		oneeSama.op = opts.fullLinks ? false : op_post.num;
		var first = oneeSama.monomono(op_post, full && 'full');
		first.pop();
		response.write(first.join(''));
		if (omit) {
			var o = common.abbrev_msg(omit, image_omit);
			if (opts.loadAllPostsLink)
				o += ' '+common.action_link_html(op_post.num,
						'See all');
			response.write('\t<span class="omit">'+o+'</span>\n');
		}
	});
	reader.on('post', function (post) {
		response.write(oneeSama.mono(post));
	});
	reader.on('endthread', function () {
		response.write('</section><hr>\n');
	});
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
	var client = STATE.clients[client_id];
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
	var path = imager.config.MEDIA_URL + 'css/';
	bits.push(['stylesheet', path + STATE.hot.BASE_CSS]);
	bits.push(['stylesheet', path + STATE.hot.BOARD_CSS[board], 'theme']);
	return bits.map(function (p) {
		var html = '\t<link rel="'+p[0]+'" href="'+p[1]+'"';
		if (p[2])
			html += ' id="' + p[2] + '"';
		return html + '>\n';
	}).join('');
}

function write_board_head(resp, board, nav) {
	var indexTmpl = RES.indexTmpl;
	var title = STATE.hot.TITLES[board] || escape(board);
	resp.write(indexTmpl[0]);
	resp.write(title);
	resp.write(indexTmpl[1]);
	resp.write(make_board_meta(board, nav));
	resp.write(indexTmpl[2]);
	if (RES.navigationHtml)
		resp.write(RES.navigationHtml);
	resp.write(indexTmpl[3]);
	resp.write(title);
	resp.write(indexTmpl[4]);
}

function write_thread_head(resp, board, op, subject, abbrev) {
	var indexTmpl = RES.indexTmpl;
	var title = '/'+escape(board)+'/ - ';
	if (subject)
		title += escape(subject) + ' (#' + op + ')';
	else
		title += '#' + op;

	resp.write(indexTmpl[0]);
	resp.write(title);
	resp.write(indexTmpl[1]);
	resp.write(make_thread_meta(board, op, abbrev));
	resp.write(indexTmpl[2]);
	if (RES.navigationHtml)
		resp.write(RES.navigationHtml);
	resp.write(indexTmpl[3]);
	resp.write('Thread #' + op);
	resp.write(indexTmpl[4]);
	resp.write(common.action_link_html('#bottom', 'Bottom'));
	resp.write('<hr>\n');
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

	// XXX in-node impl blocks node while deflating... not good!
	var gz = require('zlib').createGzip();
	gz.pipe(resp);
	return gz;
}

function make_board_meta(board, info) {
	var bits = [];
	if (info.cur_page >= 0)
		bits.push(['index', '.']);
	if (info.prev_page)
		bits.push(['prev', info.prev_page]);
	if (info.next_page)
		bits.push(['next', info.next_page]);
	return make_link_rels(board, bits);
}

function make_thread_meta(board, num, abbrev) {
	var bits = [['index', '.']];
	if (abbrev)
		bits.push(['canonical', num]);
	return make_link_rels(board, bits);
}

function make_pagination_html(info) {
	var bits = ['<nav class="pagination">'], cur = info.cur_page;
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
		bits.push(' <input type="button" value="Next">');
	bits.push('</nav>');
	return bits.join('');
}

function redirect_thread(cb, num, op, tag) {
	if (!tag)
		cb(null, 'redirect', op + '#' + num);
	else
		/* Use a JS redirect to preserve the hash */
		cb(null, 'redirect_js', '../' + tag + '/' + op + '#' + num);
}

web.route_post(/^\/upload\/$/, require('../imager/daemon').new_upload);

web.resource(/^\/$/, function (req, cb) {
	cb(null, 'redirect', config.DEFAULT_BOARD + '/');
});

web.route_post(/^\/login$/, persona.login);
web.route_post_auth(/^\/logout$/, persona.logout);
if (config.DEBUG) {
	/* Shortcuts for convenience */
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
		resp.write(RES.loginHtml[0]);
		resp.write('{}');
		resp.end(RES.loginHtml[1]);
	});

	web.resource(/^\/logout$/, function (req, cb) {
		if (req.ident.auth)
			cb(null, 'ok');
		else
			cb(null, 'redirect', config.DEFAULT_BOARD+'/');
	},
	function (req, resp) {
		resp.writeHead(200, web.noCacheHeaders);
		resp.write(RES.loginHtml[0]);
		resp.write(JSON.stringify({
			loggedInUser: req.ident.email,
			x_csrf: req.ident.csrf,
		}));
		resp.end(RES.loginHtml[1]);
	});
}
web.resource(/^\/(login|logout)\/$/, true, function (req, params, cb) {
	cb(null, 'redirect', '../' + params[1]);
});

function write_mod_js(resp, ident) {
	resp.writeHead(200, {
			'Content-Type': 'text/javascript; charset=UTF-8'});
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
		auth: 'Admin', email: req.ident.email,
		csrf: req.ident.csrf,
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
		auth: 'Moderator', email: req.ident.email,
		csrf: req.ident.csrf,
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

	cb(null, 'ok', {board: board});
},
function (req, resp) {
	/* TEMP */
	if (req.ident.suspension)
		return render_suspension(req, resp);

	var board = this.board;
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

	var yaku = new db.Yakusoku(board, req.ident);
	yaku.get_tag(0);
	var paginationHtml;
	yaku.once('begin', function (thread_count) {
		var nav = page_nav(thread_count, -1);
		write_board_head(resp, board, nav);
		paginationHtml = make_pagination_html(nav);
		resp.write(paginationHtml);
		resp.write('<hr>\n');
	});
	resp = write_gzip_head(req, resp, web.noCacheHeaders);
	var opts = {fullLinks: true, board: board};
	write_thread_html(yaku, req, resp, opts);
	yaku.once('end', function () {
		resp.write(paginationHtml);
		write_page_end(req, resp, false);
		yaku.disconnect();
	});
	yaku.once('error', function (err) {
		winston.error('index:', err);
		resp.end();
		yaku.disconnect();
	});
});

web.resource(/^\/(\w+)\/page(\d+)$/, function (req, params, cb) {
	var board = params[1];
	if (caps.under_curfew(req.ident, board))
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
		cb(404);
		yaku.disconnect();
	});
	yaku.once('begin', function (threadCount) {
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
	var nav = page_nav(this.threadCount, this.page);
	resp = write_gzip_head(req, resp, web.noCacheHeaders);
	write_board_head(resp, board, nav);
	var paginationHtml = make_pagination_html(nav);
	resp.write(paginationHtml);
	resp.write('<hr>\n');

	var opts = {fullLinks: true, board: board};
	write_thread_html(this.yaku, req, resp, opts);
	var self = this;
	this.yaku.once('end', function () {
		resp.write(paginationHtml);
		write_page_end(req, resp, false);
		self.finished();
	});
	this.yaku.once('error', function (err) {
		winston.error('page', self.page + ':', err);
		resp.end();
		self.finished();
	});
},
function () {
	this.yaku.disconnect();
});

web.resource(/^\/(\w+)\/page(\d+)\/$/, function (req, params, cb) {
	if (caps.under_curfew(req.ident, params[1]))
		cb(null, 302, '..');
	else
		cb(null, 'redirect', '../page' + params[2]);
});

web.resource(/^\/(\w+)\/(\d+)$/, function (req, params, cb) {
	var board = params[1];
	if (caps.under_curfew(req.ident, board))
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
	var op;
	if (board == 'graveyard') {
		op = num;
	}
	else {
		op = db.OPs[num];
		if (!op)
			return cb(404);
		if (!db.OP_has_tag(board, op)) {
			var tag = db.first_tag_of(op);
			if (tag) {
				if (!caps.can_access_board(req.ident, tag))
					return cb(404);
				return redirect_thread(cb, num, op, tag);
			}
			else {
				winston.warn("Orphaned post", num,
					"with tagless OP", op);
				return cb(404);
			}
		}
		if (op != num)
			return redirect_thread(cb, num, op);
	}
	if (!caps.can_access_thread(req.ident, op))
		return cb(404);

	var yaku = new db.Yakusoku(board, req.ident);
	var reader = new db.Reader(yaku);
	var lastN = config.THREAD_LAST_N;
	var opts = {redirect: true};
	if (('last'+lastN) in req.query)
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
			if (common.thumbStyles.indexOf(chunks.thumb) >= 0)
				etag += '-' + chunks.thumb;
			if (preThread.locked)
				etag += '-locked';
			if (req.ident.auth)
				etag += '-auth';
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

	var board = this.board, op = this.op;

	resp = write_gzip_head(req, resp, this.headers);
	write_thread_head(resp, board, op, this.subject, this.abbrev);

	var opts = {fullPosts: true, board: board, loadAllPostsLink: true};
	write_thread_html(this.reader, req, resp, opts);
	var self = this;
	this.reader.once('end', function () {
		write_page_end(req, resp, true);
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

web.resource(/^\/(\w+)\/(\d+)\/$/, function (req, params, cb) {
	if (caps.under_curfew(req.ident, params[1]))
		cb(null, 302, '..');
	else
		cb(null, 'redirect', '../' + params[2]);
});

var returnHTML = common.action_link_html('.', 'Return').replace(
		'span', 'span id="bottom"');

function write_page_end(req, resp, returnLink) {
	resp.write(RES.indexTmpl[5]);
	if (returnLink)
		resp.write(returnHTML);
	else if (RES.navigationHtml)
		resp.write('<br><br>' + RES.navigationHtml);
	resp.write(RES.indexTmpl[6]);
	if (req.ident) {
		if (caps.can_administrate(req.ident))
			resp.write('<script src="../admin.js"></script>\n');
		else if (caps.can_moderate(req.ident))
			resp.write('<script src="../mod.js"></script>\n');
	}
	resp.end();
}

web.resource(/^\/outbound\/(g|iqdb)\/([\w+\/]{22}\.jpg)$/,
			function (req, params, cb) {
	var thumb = imager.config.MEDIA_URL + 'vint/' + params[2];
	var service = params[1] == 'iqdb' ? 'http://iqdb.org/?url='
			: 'http://google.com/searchbyimage?image_url=';
	var dest = service + encodeURIComponent(thumb);
	cb(null, 303.1, dest);
});

web.resource(/^\/outbound\/hash\/([\w+\/]{22})$/, function (req, params, cb) {
	var dest = 'http://archive.foolz.us/search/image/' + escape(params[1]);
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

web.route_get_auth(/^\/dead\/(src|thumb)\/(\w+\.\w{3})$/,
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
			client.report(Muggle("Allocation failure.", err));
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
	var post = {time: new Date().getTime(), nonce: msg.nonce};
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
		body = msg.frag.replace(config.EXCLUDE_REGEXP, '');
		if (config.GAME_BOARDS.indexOf(client.board) >= 0)
			amusement.roll_dice(body, post, extra);
	}

	if (msg.op)
		post.op = msg.op;
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

	/* TODO: Check against client.watching? */
	if (msg.name) {
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
				client.report(Muggle("Couldn't add text.",
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
		client.report(Muggle("Failure requested."));
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
			client.report(Muggle("Couldn't finish post.", err));
	});
	return true;
}

dispatcher[common.DELETE_POSTS] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.report(Muggle("Couldn't delete."));
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
			client.report(Muggle("Couldn't delete.", err));
	});
});

dispatcher[common.LOCK_THREAD] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.report(Muggle("Couldn't (un)lock thread."));
	nums = nums.filter(function (op) { return db.OPs[op] == op; });
	async.forEach(nums, client.db.toggle_thread_lock.bind(client.db),
				function (err) {
		if (err)
			client.report(Muggle("Couldn't (un)lock thread.",err));
	});
});

dispatcher[common.DELETE_IMAGES] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.report(Muggle("Couldn't delete images."));
	client.db.remove_images(nums, function (err, dels) {
		if (err)
			client.report(Muggle("Couldn't delete images.", err));
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
			return client.report(Muggle("Image lost.", err));
		if (!client.post || client.post.image)
			return;
		client.db.add_image(client.post, alloc, client.ident.ip,
					function (err) {
			if (err)
				client.report(Muggle(
					"Image insertion problem.", err));
		});
	});
	return true;
};

dispatcher[common.SPOILER_IMAGES] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.report(Muggle("Couldn't spoiler images."));
	client.db.force_image_spoilers(nums, function (err) {
		if (err)
			client.report(Muggle("Couldn't spoiler images.", err));
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
			client.report(err);
	});
	return true;
};

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

function propagate_resources() {
	if (!tripcode.setSalt(config.SECURE_SALT))
		throw "Bad SECURE_SALT";
	web.notFoundHtml = RES.notFoundHtml;
	web.serverErrorHtml = RES.serverErrorHtml;
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
	var sockjsPath = 'js/' + get_sockjs_script_sync();
	var sockOpts = {
		sockjs_url: imager.config.MEDIA_URL + sockjsPath,
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

		var client = new okyaku.Okyaku(socket, ip);
		socket.on('data', client.on_message.bind(client));
		socket.on('close', client.on_close.bind(client));
	});

	process.on('SIGHUP', hot_reloader);

	if (config.DAEMON) {
		var cfg = config.DAEMON;
		var daemon = require('daemon');
		var pid = daemon.start(process.stdout.fd, process.stderr.fd);
		var lock = require('path').join(cfg.PID_PATH, 'server.pid');
		daemon.lock(lock);
		winston.remove(winston.transports.Console);
		return;
	}

	process.nextTick(non_daemon_pid_setup);

	winston.info('Listening on ' + (config.LISTEN_HOST || '')
			+ ':' + config.LISTEN_PORT + '.');
}

function hot_reloader() {
	async.series([
		STATE.reload_hot,
		STATE.reset_resources,
	], function (err) {
		if (err) {
			winston.error("Error trying to reload:");
			winston.error(err);
			return;
		}
		propagate_resources();
		okyaku.scan_client_caps();
		winston.info('Reloaded initial state.');
	});
}

function non_daemon_pid_setup() {
	var path = require('path');
	var pidFile = path.join(path.dirname(module.filename), '.server.pid');
	fs.writeFile(pidFile, process.pid+'\n', function (err) {
		if (err)
			return winston.warn("Couldn't write pid:", err);
		process.once('SIGINT', delete_pid);
		process.once('SIGTERM', delete_pid);
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
	async.series([
		STATE.reload_hot,
		imager.make_media_dirs,
		setup_imager_relay,
		STATE.reset_resources,
		db.track_OPs,
	], function (err) {
		if (err)
			throw err;
		propagate_resources();

		var yaku = new db.Yakusoku(null, db.UPKEEP_IDENT);
		var onegai = new imager.Onegai;
		var writes = [];
		if (!config.READ_ONLY) {
			writes.push(
				yaku.finish_all.bind(yaku),
				onegai.delete_temporaries.bind(onegai)
			);
		}
		async.series(writes, function (err) {
			if (err)
				throw err;
			yaku.disconnect();
			onegai.disconnect();
			process.nextTick(start_server);
		});
	});
}
