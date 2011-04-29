var common = require('./common'),
    config = require('./config'),
    flow = require('flow'),
    fs = require('fs'),
    io = require('socket.io'),
    http = require('http'),
    pix = require('./pix'),
    db = require('./db'),
    Template = require('./lib/json-template').Template,
    tripcode,
    util = require('util');

var clients = {};
var dispatcher = {};
var indexTmpl, notFoundHtml;

function private_msg(client, msg) {
	client.socket.send(JSON.stringify([msg]));
}

dispatcher[common.SYNCHRONIZE] = function (msg, client) {
	if (msg.length != 2)
		return false;
	var syncs = msg[0];
	if (typeof syncs != 'object')
		return false;
	/* TODO: Limit thread subscriptions */
	var dead_threads = [], count = 0;
	for (var k in syncs) {
		if (typeof syncs[k] != 'number')
			return false;
		if (db.OPs[k] != k) {
			delete syncs[k];
			dead_threads.push(k);
		}
		if (++count > config.THREADS_PER_PAGE)
			return false;
	}
	client.watching = syncs;
	/* Race between subscribe and backlog fetch... hmmm... */
	flow.exec(function () {
		client.db.kiku(client.watching, this);
	},
	function (errs) {
		if (errs && errs.length >= count)
			report("Couldn't synchronize to board.", client);
		else {
			if (errs) {
				/* XXX: warn */
			}
			client.db.fetch_backlog(client.watching, this);
		}
	},
	function (err, s, log) {
		if (err)
			return report(err, client);

		client.db.on('update', client_update.bind(client));

		if (s != sync + log.length)
			console.error("Warning: backlog count wrong");
		if (log.length) {
			log.push('[' + common.SYNCHRONIZE + ',0]');
			client.socket.send('[' + log.join() + ']');
		}
		else
			private_msg(client, [common.SYNCHRONIZE, 0]);
		client.synced = true;
	});
	return true;
}

function client_update(thread, num, kind, msg) {
	var mine = (this.post && this.post.num == num) || this.last_num == num;
	if (mine && kind != common.FINISH_POST) {
		this.skipped++;
		return;
	}
	if (this.skipped) {
		console.log("Skipping ahead " + this.skipped);
		msg = '['+common.SYNCHRONIZE+','+this.skipped+'],' + msg;
		this.skipped = 0;
	}
	this.socket.send('[' + msg + ']');
}

var oneeSama = new common.OneeSama(function (num) {
	var op = db.OPs[num];
	if (op)
		this.callback(common.safe('<a href="'
				+ common.post_url({op: op, num: num}, false)
				+ '">&gt;&gt;' + num + '</a>'));
	else
		this.callback('>>' + num);
});
oneeSama.dirs = {src_url: config.IMAGE_URL, thumb_url: config.THUMB_URL};

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
	private_msg(this.client, [common.IMAGE_STATUS, status]);
}

var httpHeaders = {'Content-Type': 'text/html; charset=UTF-8',
		'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT, -1',
		'Cache-Control': 'no-cache'};
var server = http.createServer(function(req, resp) {
	if (req.method.toLowerCase() == 'post') {
		var upload = new pix.ImageUpload(clients, allocate_post,
				image_status);
		upload.handle_request(req, resp);
		return;
	}
	if (req.url == '/' && render_index(req, resp))
		return;
	m = req.url.match(/^\/(\d+)$/);
	if (m && render_thread(req, resp, parseInt(m[1])))
		return;
	if (config.DEBUG) {
		/* Highly insecure! Abunai! */
		var path = '../www/' + req.url.replace(/\.\./g, '');
		var s = fs.createReadStream(path);
		s.once('error', function (err) {
			if (err.code == 'ENOENT')
				render_404(resp);
			else {
				resp.writeHead(500, {});
				resp.end(err.message);
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
		return;
	}
	render_404(resp);
});

function render_index(req, resp) {
	var yaku = new db.Yakusoku();
	yaku.get_tag();
	yaku.on('begin', function () {
		resp.writeHead(200, httpHeaders);
		resp.write(indexTmpl[0]);
		resp.write(config.TITLE);
		resp.write(indexTmpl[1]);
	});
	write_thread_html(yaku, resp, false);
	yaku.on('end', function () {
		yaku.get_sync_number(function (err, sync_num) {
			if (err)
				return yaku.emit('error', err);
			resp.end(indexTmpl[2]);
			yaku.disconnect();
		});
	});
	yaku.on('error', function (err) {
		console.error('index:', err);
		resp.end();
		yaku.disconnect();
	});
	return true;
}

function render_404(resp) {
	resp.writeHead(404, httpHeaders);
	resp.end(notFoundHtml);
}

function redirect_thread(resp, num, op) {
	resp.writeHead(302, {Location: op + '#' + num});
	resp.end();
}

function render_thread(req, resp, num) {
	var op = db.OPs[num];
	if (typeof op == 'undefined')
		return render_404(resp);
	if (op != num)
		return redirect_thread(resp, num, op);
	var yaku = new db.Yakusoku();
	var reader = new db.Reader(yaku);
	reader.get_thread(num, true, false);
	reader.on('nomatch', render_404.bind(null, resp));
	reader.on('redirect', redirect_thread.bind(null, resp, num));
	reader.on('begin', function () {
		resp.writeHead(200, httpHeaders);
		resp.write(indexTmpl[0]);
		resp.write('Thread #' + op);
		resp.write(indexTmpl[1]);
	});
	write_thread_html(reader, resp, true);
	reader.on('end', function () {
		resp.write('[<a href=".">Return</a>]');
		yaku.get_sync_number(function (err, sync_num) {
			if (err)
				reader.emit('error', err);
			resp.end(indexTmpl[2]);
			yaku.disconnect();
		});
	});
	function on_err(err) {
		console.error('thread '+num+':', err);
		resp.end();
		yaku.disconnect();
	}
	reader.on('error', on_err);
	yaku.on('error', on_err);
	return true;
}

function on_client (socket, retry) {
	if (socket.connection)
		init_client(socket);
	else if (!retry || retry < 5000) {
		/* Wait for socket.connection */
		retry = retry ? retry*2 : 50;
		setTimeout(function () {
			on_client(socket, retry);
		}, retry);
	}
	else
		util.error("Dropping no-connection client (?!)");
}

function init_client (socket) {
	var ip = socket.connection.remoteAddress;
	var id = socket.sessionId;
	console.log(id + " has IP " + ip);
	var client = {id: id, socket: socket, post: null, synced: false,
			watching: {}, ip: ip, db: new db.Yakusoku(),
			skipped: 0};
	clients[id] = client;
	socket.on('message', function (data) {
		var msg = null;
		try { msg = JSON.parse(data); }
		catch (e) {}
		var type = common.INVALID;
		if (msg == null) {
		}
		else if (client.post && msg.constructor == String)
			type = common.UPDATE_POST;
		else if (msg.constructor == Array)
			type = msg.shift();
		var func = dispatcher[type];
		if (!func || !func(msg, client)) {
			console.error("Got invalid message " + data);
			report(null, client, "Bad protocol.");
		}
	});
	socket.on('disconnect', function () {
		delete clients[id];
		client.synced = false;
		if (client.post)
			finish_post_by(client, function () {
				client.db.disconnect();
			});
		else
			client.db.disconnect();
	});
	socket.on('error', console.error.bind(console, 'socket:'));
	client.db.on('error', console.error.bind(console, 'redis:'));
}

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
			private_msg(client, [common.INVALID, msg + ver]);
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
	if (typeof msg != 'object' || !msg.op)
		return false;
	if (client.post)
		return update_post(msg.frag, client);
	var frag = msg.frag;
	if (!frag || frag.match(/^\s*$/g))
		return false;
	allocate_post(msg, null, client, function (err, alloc) {
		if (err)
			return report(err, client, "Couldn't allocate post.");
		var go = private_msg.bind(null, client,
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
		if (post.email == 'noko')
			delete post.email;
	}
	if (image)
		post.image = image;
	post.state = [0, 0];
	flow.exec(function () {
		client.db.reserve_post(post.op, this);
	},
	function (err, num) {
		if (err)
			return callback("Couldn't reserve a post.");
		if (client.post)
			return callback('Already have a post.');
		client.post = post;
		post.num = num;
		valid_links(body, post.state, this);
	},
	function (err, links) {
		if (err) {
			console.error('valid_links: ' + err);
			if (client.post === post)
				delete client.post;
			return callback("Post reference error.");
		}
		post.links = links;
		client.db.insert_post(post, body, client.ip, this);
	},
	function (err) {
		if (err) {
			if (client.post === post)
				delete client.post;
			console.error(err);
			return callback("Couldn't allocate post.");
		}
		post.body = body;
		callback(null, get_post_view(post));
	});
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
	flow.exec(function () {
		valid_links(frag, post.state, this);
	},
	function (err, links) {
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
				this);
	},
	function (err) {
		if (err)
			report(err, client, "Couldn't add text.");
	});
	return true;
}
dispatcher[common.UPDATE_POST] = update_post;

function finish_post_by(client, callback) {
	/* TODO: Should we check client.uploading? */
	client.db.finish_post(client.post, function (err) {
		if (err)
			callback(err);
		else {
			client.last_num = client.post.num;
			delete client.post;
			callback(null);
		}
	});
}

dispatcher[common.FINISH_POST] = function (msg, client) {
	if (msg.length || !client.post)
		return false;
	finish_post_by(client, function (err) {
		if (err)
			report(err, client, "Couldn't finish post.");
	});
	return true;
}

function start_server() {
	server.listen(config.PORT);
	var listener = io.listen(server, {
		transports: ['websocket', 'jsonp-polling', 'htmlfile',
			'xhr-multipart', 'xhr-polling']
	});
	listener.on('connection', on_client);
	listener.on('error', function (err) {
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
		indexTmpl = Template(fs.readFileSync('index.html', 'UTF-8'),
			{meta: '{{}}'}).expand(config).split(/\$[A-Z]+/);
		notFoundHtml = fs.readFileSync('../www/404.html');
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
