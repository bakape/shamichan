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

function multisend(client, msgs) {
	client.socket.send(JSON.stringify(msgs));
}

dispatcher[common.SYNCHRONIZE] = function (msg, client) {
	if (msg.length != 2)
		return false;
	var sync = msg[0], watching = msg[1];
	if (typeof sync != 'number' || sync < 0 || isNaN(sync))
		return false;
	if (watching) {
		if (watching.constructor != Number)
			return false;
		client.db.thread_exists(watching, function (err, exists) {
			if (err)
				report(err, client);
			else if (!exists)
				report(null, client, "No such thread.");
			else {
				client.watching = watching;
				sync_client(client, sync);
			}
		});
	}
	else
		sync_client(client, sync);
	return true;
};

function sync_client(client, sync) {
	/* Race between subscribe and backlog fetch... hmmm... */
	flow.exec(function () {
		client.db.kiku(client.watching, this);
	},
	function (err) {
		if (err)
			report(err, client);
		else
			client.db.fetch_backlog(sync, client.watching, this);
	},
	function (err, s, log) {
		if (err)
			return report(err, client);

		client.db.on('update', client_update.bind(client));

		if (log.length == 0)
			multisend(client, [[common.SYNCHRONIZE, s]]);
		else {
			log.push('[' + common.SYNCHRONIZE + ',' + s + ']');
			client.socket.send('[' + log.join() + ']');
		}
		client.synced = true;
	});
}

function client_update(thread, num, kind, msg) {
	if (this.post && this.post.num == num && kind != common.FINISH_POST) {
		/* TODO: Synchronize them instead */
		return;
	}
	this.socket.send('[' + msg + ']');
}

var oneeSama = new common.OneeSama(function (num) {
	var post = posts[num];
	if (post)
		this.callback(common.safe('<a href="'
				+ common.post_url(post, false)
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
	multisend(this.client, [[common.IMAGE_STATUS, status]]);
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
	if (m && render_thread(req, resp, m[1]))
		return;
	if (config.DEBUG) {
		/* Highly insecure! Abunai! */
		var path = '../www/' + req.url.replace(/\.\./g, '');
		var s = fs.createReadStream(path);
		s.once('error', function (err) {
			if (err.code == 'ENOENT') {
				resp.writeHead(404, httpHeaders);
				resp.end(notFoundHtml);
			}
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
	resp.writeHead(404, httpHeaders);
	resp.end(notFoundHtml);
});

function render_index(req, resp) {
	var yaku = new db.Yakusoku();
	yaku.get_tag();
	yaku.on('begin', function () {
		resp.writeHead(200, httpHeaders);
		resp.write(indexTmpl[0]);
	});
	write_thread_html(yaku, resp, false);
	yaku.on('end', function () {
		yaku.get_sync_number(function (err, sync_num) {
			if (err)
				return yaku.emit('error', err);
			resp.write(indexTmpl[1]);
			resp.write(''+sync_num);
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

function render_thread(req, resp, num) {
	var yaku = new db.Yakusoku();
	var reader = new db.Reader(yaku);
	reader.get_thread(parseInt(num), true, false);
	reader.on('nomatch', function () {
		resp.writeHead(404, httpHeaders);
		resp.end(notFoundHtml);
	});
	reader.on('redirect', function (op) {
		resp.writeHead(302, {Location: op + '#' + num});
		resp.end();
	});
	reader.on('begin', function () {
		resp.writeHead(200, httpHeaders);
		resp.write(indexTmpl[0]);
	});
	write_thread_html(reader, resp, true);
	reader.on('end', function () {
		resp.write('[<a href=".">Return</a>]');
		yaku.get_sync_number(function (err, sync_num) {
			if (err)
				reader.emit('error', err);
			resp.write(indexTmpl[1]);
			resp.write(''+sync_num);
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
			watching: null, ip: ip, db: new db.Yakusoku()};
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
			multisend(client, [[common.INVALID, msg + ver]]);
			client.synced = false;
		}
	});
}

/* Must be prepared to receive callback instantly */
function valid_links(frag, state, callback) {
	var possible = {}, links = {};
	var checked = 0, total = 0;
	var done = false;
	function got_post_op(err, num, op) {
		if (done)
			return;
		if (err) {
			done = true;
			return callback(err);
		}
		if (num)
			links[num] = op || num;
		if (++checked >= total) {
			done = true;
			callback(null, isEmpty(links) ? null : links);
		}
	}
	var onee = new common.OneeSama(function (num, e) {
		if (!(num in possible)) {
			total++;
			possible[num] = null;
			client.db.get_post_op(num, got_post_op);
		}
	});
	onee.callback = function (frag) {};
	onee.state = state;
	onee.fragment(frag);
	if (!total)
		callback(null, null);
}

function isEmpty(obj) {
	for (k in obj)
		if (obj.hasOwnProperty(k))
			return false;
	return true;
}

dispatcher[common.ALLOCATE_POST] = function (msg, client) {
	if (msg.length != 1)
		return false;
	msg = msg[0];
	if (!msg.op)
		return false;
	if (client.post) {
		/* TODO: merge with image upload's alloc */
		return true;
	}
	var frag = msg.frag;
	if (!frag || frag.match(/^\s*$/g))
		return false;
	allocate_post(msg, null, client, function (err, alloc) {
		if (err)
			return report(err, client, "Couldn't allocate post.");
		else
			multisend(client, [[common.ALLOCATE_POST, alloc]]);
	});
	return true;
}

function allocate_post(msg, image, client, callback) {
	if (!msg || typeof msg != 'object')
		return callback('Bad alloc.');
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
	if (client.watching && post.op !== client.watching)
		return false;
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
		valid_links(body, post.state, this);
	},
	function (err, links) {
		if (err) {
			console.error('valid_links: ' + err);
			return callback("Post reference error.");
		}
		if (client.post)
			return callback('Already have a post.');
		post.links = links;
		client.db.insert_post(post, body, client.ip, function (num) {
			post.num = num;
			post.body = body;
			client.post = post;
		}, this);
	},
	function (err) {
		if (err) {
			console.error(err);
			return callback("Couldn't allocate post.");
		}
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

dispatcher[common.UPDATE_POST] = function (frag, client) {
	if (!frag || frag.constructor != String)
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
		if (links)
			for (var k in links)
				post.links[k] = links[k];
		client.db.append_post(post, frag, old_state, links, this);
	},
	function (err) {
		if (err)
			report(err, client, "Couldn't add text.");
	});
	return true;
}

function finish_post_by(client, callback) {
	/* TODO: Should we check client.uploading? */
	client.db.finish_post(client.post, function (err) {
		if (err)
			callback(err);
		else {
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
	var val = config[process.argv[3]];
	if (!val)
		throw "No such config value " + process.argv[3];
	console.log(val.join ? val.join(' ') : val);
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
