/*
Core server module and application entry point
 */

// Several modules depend on the state module, so load it first
const STATE = require('./state');

const _ = require('underscore'),
    amusement = require('./amusement'),
    async = require('async'),
    caps = require('./caps'),
    check = require('./msgcheck'),
    common = require('../common/index'),
	config = require('../config'),
	cookie = require('cookie'),
	db = require('../db'),
    fs = require('fs'),
    hooks = require('../util/hooks'),
    imager = require('../imager'),
    Muggle = require('../util/etc').Muggle,
	net = require('net'),
    okyaku = require('./okyaku'),
	path = require('path'),
    persona = require('./persona'),
    winston = require('winston');

require('../imager/daemon'); // preload and confirm it works

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
	var onegai = new imager.ClientController;
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

dispatcher[common.INSERT_POST] = (msg, client) => {
	const insertSpec = [{
		frag: 'opt string',
		image: 'opt string',
		nonce: 'string',
		op: 'opt id',
		name: 'opt string',
		email: 'opt string',
		auth: 'opt string',
		subject: 'opt string'
	}]
	if (!check(insertSpec, msg))
		return false
	msg = msg[0]
	const {frag} = msg
	if (client.post)
		return update_post(frag, client)

	if (!caps.can_access_board(client.ident, client.board)
		|| (frag && /^\s*$/g.test(frag))
		|| (!frag && !msg.image)
	)
		return false

	client.db.insertPost(msg).catch(err =>
		client.kotowaru(Muggle('Allocation failure', err)))
	return true
}

function update_post(frag, client) {
	if (typeof frag !== 'string')
		return false
	frag = amusement.hot_filter(frag.replace(STATE.hot.EXCLUDE_REGEXP, ''))
	const {post} = client
	if (!post)
		return false
	const limit = common.MAX_POST_CHARS
	if (frag.length > limit || post.length  >= limit)
		return false
	const combined = post.length + frag.length
	if (combined > limit)
		frag = frag.substr(0, combined - limit)
	client.db.appendPost(frag).catch(err =>
		client.kotowaru(Muggle("Couldn't add text.", err)))
	return true
}
dispatcher[common.UPDATE_POST] = update_post;

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
		return false
	if (!client.post || client.post.image)
		return false
	client.db.insertImage(msg[0]).catch(err =>
		client.kotowaru(Muggle('Image insertion error:', err)))

	imager.obtain_image_alloc(alloc, function (err, alloc) {
		if (err)
			return client.kotowaru(Muggle("Image lost.", err))
		if (!client.post || client.post.image)
			return
		client.db.add_image(client.post, alloc, client.ident.ip,
			function (err) {
				if (err)
					client.kotowaru(Muggle("Image insertion problem.", err))
			}
		)
	})
	return true
}


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
			onegai = new imager.ClientController;
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
