/*
Core server module and application entry point
 */

// Several modules depend on the state module and a redis connection. Load
// those first.
const STATE = require('./state'),
	db = require('../db');

const _ = require('underscore'),
    amusement = require('./amusement'),
    async = require('async'),
    caps = require('./caps'),
    check = require('./msgcheck'),
    common = require('../common/index'),
	config = require('../config'),
	cookie = require('cookie'),
    fs = require('fs'),
    hooks = require('../util/hooks'),
    imager = require('../imager'),
    Muggle = require('../util/etc').Muggle,
	net = require('net'),
    okyaku = require('./okyaku'),
	path = require('path'),
    persona = require('./persona'),
    Render = require('./render'),
    tripcode = require('bindings')('tripcode'),
    urlParse = require('url').parse,
    winston = require('winston');

require('../imager/daemon'); // preload and confirm it works
var radio;
if (config.RADIO)
	radio = require('./radio');

try {
	if (config.RECAPTCHA_PUBLIC_KEY)
		require('./report');
}
catch (e) {}
require('./time');

let dispatcher = okyaku.dispatcher;

dispatcher[common.SYNCHRONIZE] = function (msg, client) {
	const personaCookie = persona.extract_login_cookie(cookie.parse(msg.pop()));
	if (personaCookie) {
		persona.check_cookie(personaCookie, function (err, ident) {
			if (!err)
				_.extend(client.ident, ident);
			if (!synchronize(msg, client))
				client.kotowaru(Muggle("Bad protocol"));
		});
		return true;
	}
	else
		return synchronize(msg, client);
};

function synchronize(msg, client) {
	if (!check(['id', 'string', 'id=>nat', 'boolean'], msg))
		return false;
	const id = msg[0];
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
	return linkToDatabase(msg[1], msg[2], msg[3], client);
}

// Establish database liteners and handlers for the client
function linkToDatabase(board, syncs, live, client) {
	if (!caps.can_access_board(client.ident, board))
		return false;
	if (client.db)
		client.db.kikanai().disconnect();

	let count = 0, op;
	for (let k in syncs) {
		k = parseInt(k, 10);
		if (!db.validateOP(k, board)) {
			delete syncs[k];
		}
		op = k;
		if (++count > STATE.hot.THREADS_PER_PAGE) {
			/* Sync logic isn't great yet; allow this for now */
			// return false;
		}
	}
	client.watching = syncs;
	if (live) {
		/* XXX: This will break if a thread disappears during sync
		 * (won't be reported)
		 * Or if any of the threads they see on the first page
		 * don't show up in the 'live' pub for whatever reason.
		 * Really we should get them synced first and *then* switch
		 * to the live pub.
		 */
		client.watching = {live: true};
		count = 1;
	}
	client.board = board;
	client.db = new db.Yakusoku(board, client.ident);
	// Race between subscribe and backlog fetch; client must de-dup
	client.db.kiku(
		client.watching,
		client.on_update.bind(client),
		client.on_thread_sink.bind(client),
		listening
	);

	function listening(errs) {
		if (errs && errs.length >= count)
			return client.kotowaru(Muggle("Couldn't sync to board."));
		else if (errs) {
			for (let err of errs) {
				delete client.watching[err];
			}
		}
		client.db.fetch_backlogs(client.watching, got_backlogs);
	}

	function got_backlogs(errs, logs) {
		if (errs) {
			for (let err of errs) {
				delete client.watching[err];
			}
		}
		for (let log of logs) {
			client.socket.write(`[[${log}]]`)
		}
		client.send([0, common.SYNCHRONIZE])
		client.synced = true;

		let info = {
			client: client,
			live: live
		};
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

// Switch the serverside syncs, when client switches them with HTML5 History
dispatcher[common.RESYNC] = function(msg, client) {
	if (!check(['string', 'id=>nat', 'boolean'], msg))
		return false;
	return linkToDatabase(msg[0], msg[1], msg[2], client);
};

// Stop listening on redis channels in preparation for RESYNC
dispatcher[common.DESYNC] = function (msg, client) {
	if (client.db)
		client.db.kikanai().disconnect();
	return true;
};

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


/* Must be prepared to receive callback instantly */
function valid_links(frag, state, ident, callback) {
	let links = {};
	let onee = new common.OneeSama({
		state,
		callback() {},
		tamashii(num) {
			const op = db.OPs[num];
			if (op && caps.can_access_thread(ident, op))
				links[num] = db.OPs[num];
		}
	});
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
	subject: 'opt string'
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

	allocate_post(msg, client, err =>
		err && client.kotowaru(Muggle("Allocation failure.", err)));
	return true;
};

function allocate_post(msg, client, callback) {
	if (client.post)
		return callback(Muggle("Already have a post."));
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
		body = hot_filter(msg.frag.replace(STATE.hot.EXCLUDE_REGEXP, ''));
	}

	if (msg.op) {
		if (!db.validateOP(msg.op, extra.board))
			return callback(Muggle('Thread does not exist.'));
		post.op = msg.op;
	}
	else {
		if (!image_alloc)
			return callback(Muggle('Image missing.'));
		var subject = (msg.subject || '').trim();
		subject = subject.replace(STATE.hot.EXCLUDE_REGEXP, '');
		subject = subject.replace(/[「」]/g, '');
		subject = subject.slice(0, STATE.hot.SUBJECT_MAX_LENGTH);
		if (subject)
			post.subject = subject;
	}

	// Replace names, when a song plays on r/a/dio
	if (radio && radio.name)
		post.name = radio.name;
	else if (client.ident.auth === 'dj' || !STATE.hot.forced_anon) {
		/* TODO: Check against client.watching? */
		if (msg.name) {
			const parsed = common.parse_name(msg.name);
			post.name = parsed[0];
			const spec = STATE.hot.SPECIAL_TRIPCODES;
			if (spec && parsed[1] && parsed[1] in spec)
				post.trip = spec[parsed[1]];
			else if (parsed[1] || parsed[2]) {
				const trip = tripcode.hash(parsed[1], parsed[2]);
				if (trip)
					post.trip = trip;
			}
		}
		if (msg.email)
			post.email = msg.email.trim().substr(0, 320);
	}

	post.state = [common.S_BOL, 0];

	if ('auth' in msg) {
		if (!msg.auth || !client.ident || msg.auth !== client.ident.auth)
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

		amusement.roll_dice(body, post);
		client.post = post;
		post.num = num;
		var supplements = {
			links: valid_links.bind(null, body, post.state, client.ident)
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
	frag = hot_filter(frag.replace(STATE.hot.EXCLUDE_REGEXP, ''));
	var post = client.post;
	if (!post)
		return false;
	var limit = common.MAX_POST_CHARS;
	if (frag.length > limit || post.length >= limit)
		return false;
	var combined = post.length + frag.length;
	if (combined > limit)
		frag = frag.substr(0, combined - limit);
	let extra = {ip: client.ident.ip};
	amusement.roll_dice(frag, extra);
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
			for (let k in links) {
				let link = links[k];
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
};

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
					client.kotowaru(Muggle("Image insertion problem.", err));
			}
		);
	});
	return true;
};


// Online count
hooks.hook('clientSynced', function(info, cb){
	info.client.send([
		0,
		common.ONLINE_COUNT,
		Object.keys(STATE.clientsByIP).length
	]);
	cb(null);
});

STATE.emitter.on('change:clientsByIP', function(){
	okyaku.push([
		0,
		common.ONLINE_COUNT,
		Object.keys(STATE.clientsByIP).length
	]);
});

// Update hot client variables on client request
dispatcher[common.HOT_INJECTION] = function(msg, client){
	if (!check(['boolean'], msg) || msg[0] !== true)
		return false;
	client.send([0, common.HOT_INJECTION, 1, STATE.clientConfigHash,
		STATE.clientHotConfig]);
	return true;
};

// Send current hot hash to client on sync
hooks.hook('clientSynced', function(info, cb){
	info.client.send([0, common.HOT_INJECTION, 0, STATE.clientConfigHash]);
	cb(null);
});

// Regex replacement filter
function hot_filter(frag) {
	let filter = STATE.hot.FILTER;
	if (!filter)
		return frag;
	for (let i =0, len = filter.length; i < len; i++) {
		let f = filter[i];
		const m = frag.match(f.p);
		if (m){
			// Case sensitivity
			if (m[0].length > 2){
				if (/[A-Z]/.test(m[0].charAt(1)))
					f.r = f.r.toUpperCase();
				else if (/[A-Z]/.test(m[0].charAt(0)))
					f.r = f.r.charAt(0).toUpperCase()+f.r.slice(1);
			}
			return frag.replace(f.p, f.r);
		}
	}
	return frag;
}

function start_server() {
	var is_unix_socket = (typeof config.LISTEN_PORT == 'string');
	if (is_unix_socket) {
		try {
			fs.unlinkSync(config.LISTEN_PORT);
		}
		catch (e) {}
	}
	// Start web server
	require('./web');
	// Start thread deletion module
	if (config.PRUNE)
		require('./prune');
	if (is_unix_socket)
		fs.chmodSync(config.LISTEN_PORT, '777'); // TEMP

	process.on('SIGHUP', hot_reloader);
	db.on_pub('reloadHot', hot_reloader);

	// Read global push messages from `scripts/send.js` and dispatch to all
	// clients
	db.on_pub('push', (chan, msg) => okyaku.push(JSON.parse(msg)));

	process.nextTick(processFileSetup);

	winston.info('Listening on '
		+ (config.LISTEN_HOST || '')
		+ (is_unix_socket ? '' : ':')
		+ (config.LISTEN_PORT + '.'));
}

function hot_reloader() {
	STATE.reload_hot_resources(function (err) {
		if (err)
			return winston.error('Error trying to reload:', err);
		okyaku.scan_client_caps();
		amusement.pushJS();
		// Push new hot variable hash to all clients
		okyaku.push([0, common.HOT_INJECTION, false, STATE.clientConfigHash]);
		winston.info('Reloaded initial state.');
	});
}

function processFileSetup() {
	const pidFile = config.PID_FILE;
	fs.writeFile(pidFile, process.pid+'\n', function (err) {
		if (err)
			return winston.warn("Couldn't write pid: ", err);
		process.once('SIGINT', deleteFiles);
		process.once('SIGTERM', deleteFiles);
		winston.info(`PID ${process.pid} written in ${pidFile}`);
	});

	function deleteFiles() {
		try {
			fs.unlinkSync(pidFile);
		}
		catch (e) {}
		process.exit();
	}
}

if (!tripcode.setSalt(config.SECURE_SALT))
	throw "Bad SECURE_SALT";
async.series(
	[
		imager.make_media_dirs,
		setup_imager_relay,
		STATE.reload_hot_resources,
		db.track_OPs
	],
	function (err) {
		if (err)
			throw err;
		var yaku = new db.Yakusoku(null, db.UPKEEP_IDENT);
		var onegai;
		var writes = [];
		if (!config.READ_ONLY) {
			writes.push(yaku.finish_all.bind(yaku));
			onegai = new imager.Onegai;
			writes.push(onegai.delete_temporaries.bind(onegai));
		}
		async.series(writes, function (err) {
			if (err)
				throw err;
			yaku.disconnect();
			process.nextTick(start_server);
		});
	}
);
