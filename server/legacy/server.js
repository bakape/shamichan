/*
 Old logic from `server/server.js`. Kept here until fully ported.
 */

function page_nav(thread_count, cur_page, ascending) {
	var page_count = Math.ceil(thread_count / STATE.hot.THREADS_PER_PAGE);
	page_count = Math.max(page_count, 1);
	return {pages: page_count, threads: thread_count,
		cur_page: cur_page, ascending: ascending};
}

function write_gzip_head(req, resp, headers) {
	var encoding = config.GZIP && req.headers['accept-encoding'];
	if (req.ident.slow || !encoding || encoding.indexOf('gzip') < 0) {
		resp.writeHead(200, headers);
		return resp;
	}
	resp.writeHead(200, _.extend({}, headers, {
		'Content-Encoding': 'gzip',
		Vary: 'Accept-Encoding'
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
				x_csrf: req.ident.csrf
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
			email: req.ident.email
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
			email: req.ident.email
		});
	});

web.resource(/^\/(\w+)$/, function (req, params, cb) {
	var board = params[1];
	/* If arbitrary boards were allowed, need to escape this: */
	var dest = board + '/';
	if (req.ident.suspension)
		return cb(null, 'redirect', dest); /* TEMP */
	if (!caps.can_access_board(req.ident, board))
		return cb(404);
	cb(null, 'redirect', dest);
});

web.resource(/^\/(\w+)\/$/,
	function (req, params, cb) {
		const board = params[1];
		if (req.ident.suspension)
			return cb(null, 'ok'); /* TEMP */
		if (!caps.can_access_board(req.ident, board))
			return cb(404);

		// we don't do board etags yet
		let info = {etag: 'dummy', req: req};
		hooks.trigger_sync('buildETag', info);

		cb(null, 'ok', {board: board});
	},
	function (req, resp) {
		/* TEMP */
		if (req.ident.suspension)
			return render_suspension(req, resp);

		const board = this.board;

		let yaku = new db.Yakusoku(board, req.ident);
		yaku.get_tag(-1);
		resp = write_gzip_head(req, resp, web.noCacheHeaders);
		new Render(yaku, req, resp, {
			fullLinks: true,
			board: board,
			isThread: false
		});
		yaku.once('begin', function (thread_count) {
			yaku.emit('top', page_nav(thread_count, -1, board == 'archive'));
		});
		yaku.once('end', function () {
			yaku.emit('bottom');
			resp.end();
			yaku.disconnect();
		});
		yaku.once('error', function (err) {
			winston.error('index:' + err);
			resp.end();
			yaku.disconnect();
		});
	}
);

web.resource(/^\/(\w+)\/page(\d+)$/,
	function (req, params, cb) {
		const board = params[1];
		if (!caps.temporal_access_check(req.ident, board))
			return cb(null, 302, '..');
		if (req.ident.suspension)
			return cb(null, 'ok'); /* TEMP */
		if (!caps.can_access_board(req.ident, board))
			return cb(404);
		const page = parseInt(params[2], 10);
		if (page > 0 && params[2][0] == '0') /* leading zeroes? */
			return cb(null, 'redirect', 'page' + page);

		let yaku = new db.Yakusoku(board, req.ident);
		yaku.get_tag(page);
		yaku.once('nomatch', function () {
			cb(null, 302, '.');
			yaku.disconnect();
		});
		yaku.once('begin', function (threadCount) {
			// we don't do board etags yet
			let info = {etag: 'dummy', req: req};
			hooks.trigger_sync('buildETag', info);

			cb(null, 'ok', {
				board: board, page: page, yaku: yaku,
				threadCount: threadCount
			});
		});
	},
	function (req, resp) {
		/* TEMP */
		if (req.ident.suspension)
			return render_suspension(req, resp);

		const board = this.board;
		const nav = page_nav(this.threadCount, this.page, board == 'archive');
		resp = write_gzip_head(req, resp, web.noCacheHeaders);
		new Render(this.yaku, req, resp, {
			fullLinks: true,
			board: board,
			isThread: false
		});
		this.yaku.emit('top', nav);
		let self = this;
		this.yaku.once('end', function () {
			self.yaku.emit('bottom');
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
	}
);

web.resource(/^\/(\w+)\/page(\d+)\/$/, function (req, params, cb) {
	if (!caps.temporal_access_check(req.ident, params[1]))
		cb(null, 302, '..');
	else
		cb(null, 'redirect', '../page' + params[2]);
});

web.resource(/^\/(\w+)\/(\d+)$/,
	function (req, params, cb) {
		const board = params[1];
		if (!caps.temporal_access_check(req.ident, board))
			return cb(null, 302, '.');
		if (req.ident.suspension)
			return cb(null, 'ok'); /* TEMP */
		if (!caps.can_access_board(req.ident, board))
			return cb(404);
		const num = parseInt(params[2], 10);
		if (!num)
			return cb(404);
		else if (params[2][0] == '0')
			return cb(null, 'redirect', '' + num);

		let op;
		if (board === 'graveyard') {
			op = num;
		}
		else {
			op = db.OPs[num];
			if (!op)
				return cb(404);
			if (!db.OP_has_tag(board, op)) {
				let tag = db.first_tag_of(op);
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
			if (op != num)
				return redirect_thread(cb, num, op);
		}
		if (!caps.can_access_thread(req.ident, op))
			return cb(404);

		let yaku = new db.Yakusoku(board, req.ident),
			reader = new db.Reader(yaku),
			opts = {redirect: true};

		const lastN = detect_last_n(req.query);
		if (lastN)
			opts.abbrev = lastN + STATE.hot.ABBREVIATED_REPLIES;

		if (caps.can_administrate(req.ident) && 'reported' in req.query)
			opts.showDead = true;
		reader.get_thread(board, num, opts);
		reader.once('nomatch', function() {
			cb(404);
			yaku.disconnect();
		});
		reader.once('redirect', function(op) {
			redirect_thread(cb, num, op);
			yaku.disconnect();
		});
		reader.once('begin', function(preThread) {
			let headers;
			if (!config.DEBUG && preThread.hctr) {
				// XXX: Always uses the hash of the default language in the etag
				let etag = `W/${preThread.hctr}-`
					+ RES['indexHash-' + config.DEFAULT_LANG];
				const chunks = web.parse_cookie(req.headers.cookie),
					thumb = req.cookies.thumb;
				if (thumb && common.thumbStyles.indexOf(thumb) >= 0)
					etag += '-' + thumb;
				const etags = ['spoil', 'agif', 'rtime', 'linkify', 'lang'];
				for (let i = 0, l = etags.length; i < l; i++) {
					let tag = etags[i];
					if (chunks[tag])
						etag += `-${tag}-${chunks[tag]}`;
				}
				if (lastN)
					etag += '-last' + lastN;
				if (preThread.locked)
					etag += '-locked';
				if (req.ident.auth)
					etag += '-auth';

				let info = {etag: etag, req: req};
				hooks.trigger_sync('buildETag', info);
				etag = info.etag;

				if (req.headers['if-none-match'] === etag) {
					yaku.disconnect();
					return cb(null, 304);
				}
				headers = _.clone(web.vanillaHeaders);
				headers.ETag = etag;
				headers['Cache-Control'] = 'private, max-age=0, must-revalidate';
			}
			else
				headers = web.noCacheHeaders;

			cb(null, 'ok', {
				headers: headers,
				board: board, op: op,
				subject: preThread.subject,
				yaku: yaku, reader: reader,
				abbrev: opts.abbrev
			});
		});
	},
	function (req, resp) {
		/* TEMP */
		if (req.ident.suspension)
			return render_suspension(req, resp);

		resp = write_gzip_head(req, resp, this.headers);
		new Render(this.reader, req, resp, {
			fullPosts: true,
			board: this.board,
			op: this.op,
			subject: this.subject,
			isThread: true
		});
		this.reader.emit('top');
		var self = this;
		this.reader.once('end', function () {
			self.reader.emit('bottom');
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
	function() {
		this.yaku.disconnect();
	}
);

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

web.route_get_auth(/^\/dead\/(src|thumb|mid)\/(\w+\.\w{3})$/,
	function (req, resp, params) {
		if (!caps.can_administrate(req.ident))
			return web.render_404(resp);
		imager.send_dead_image(params[1], params[2], resp);
	});
