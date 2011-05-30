var common = require('./common'),
    config = require('./config'),
    express = require('express'),
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

function Okyaku() {
}
var OK = Okyaku.prototype;

OK.send = function (msg) {
	this.socket.send(JSON.stringify([msg]));
};

dispatcher[common.SYNCHRONIZE] = function (msg, client) {
	if (msg.length != 2)
		return false;
	var syncs = msg[0];
	if (typeof syncs != 'object')
		return false;
	if (client.synced) {
		console.error("warning: Client tried to sync twice");
		return false;
	}
	var dead_threads = [], count = 0;
	for (var k in syncs) {
		if (!k.match(/^\d+$/))
			return false;
		k = parseInt(k);
		if (!k || typeof syncs[k] != 'number')
			return false;
		if (db.OPs[k] != k) {
			delete syncs[k];
			dead_threads.push(k);
		}
		if (++count > config.THREADS_PER_PAGE)
			return false;
	}
	client.watching = syncs;
	/* Race between subscribe and backlog fetch; client must de-dup */
	flow.exec(function () {
		client.db.kiku(client.watching, client.on_update.bind(client),
				client.on_thread_sink.bind(client), this);
	},
	function (errs) {
		if (errs && errs.length >= count)
			return report("Couldn't sync to board.", client);
		else if (errs) {
			dead_threads.push.apply(dead_threads, errs);
			errs.forEach(function (thread) {
				delete client.watching[thread];
			});
		}
		client.db.fetch_backlogs(client.watching, this);
	},
	function (errs, logs) {
		if (errs) {
			dead_threads.push.apply(dead_threads, errs);
			errs.forEach(function (thread) {
				delete client.watching[thread];
			});
		}

		logs.push([common.SYNCHRONIZE, dead_threads]);
		client.socket.send(JSON.stringify(logs));
		client.synced = true;
	});
	return true;
}

OK.on_update = function(op, num, kind, msg) {
	var mine = (this.post && this.post.num == num) || this.last_num == num;
	if (mine && kind != common.FINISH_POST) {
		this.skipped++;
		return;
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
	this.client.send([common.IMAGE_STATUS, status]);
}

function make_nav_html(page_count, cur_page) {
	var bits = ['<nav>'];
	if (cur_page >= 0)
		bits.push('<a href="live">live</a>');
	else
		bits.push('<b>live</b>');
	for (var i = 0; i < page_count; i++) {
		if (i != cur_page)
			bits.push('<a href="page' + i + '">' + i + '</a>');
		else
			bits.push('<b>' + i + '</b>');
	}
	bits.push('</nav>');
	return bits.join('');
}

var server = express.createServer();
if (config.DEBUG) {
	server.use(express.staticProvider(
			require('path').join(__dirname, '..', 'www')));
}

var httpHeaders = {'Content-Type': 'text/html; charset=UTF-8',
		'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT, -1',
		'Cache-Control': 'no-cache'};

server.post('/img', function (req, resp) {
	var upload = new pix.ImageUpload(clients, allocate_post, image_status);
	upload.handle_request(req, resp);
});

server.get('/', function (req, resp) {
	resp.redirect('live', 302);
});

server.get('/live', function (req, resp) {
	var yaku = new db.Yakusoku();
	yaku.get_tag(0);
	var nav_html;
	yaku.on('begin', function (pages) {
		resp.writeHead(200, httpHeaders);
		resp.write(indexTmpl[0]);
		resp.write(config.TITLE);
		resp.write(indexTmpl[1]);
		nav_html = make_nav_html(pages, -1);
		resp.write(nav_html);
		resp.write('<hr>\n');
	});
	write_thread_html(yaku, resp, false);
	yaku.on('end', function () {
		resp.write(nav_html);
		resp.end(indexTmpl[2]);
		yaku.disconnect();
	});
	yaku.on('error', function (err) {
		console.error('index:', err);
		resp.end();
		yaku.disconnect();
	});
	return true;
});

server.get('/page:page', function (req, resp) {
	var yaku = new db.Yakusoku();
	var page = parseInt(req.param('page'));
	yaku.get_tag(page);
	yaku.on('nomatch', render_404.bind(null, resp));
	var nav_html;
	yaku.on('begin', function (pages) {
		resp.writeHead(200, httpHeaders);
		resp.write(indexTmpl[0]);
		resp.write(config.TITLE);
		resp.write(indexTmpl[1]);
		nav_html = make_nav_html(pages, page);
		resp.write(nav_html);
		resp.write('<hr>\n');
	});
	write_thread_html(yaku, resp, false);
	yaku.on('end', function () {
		resp.write(nav_html);
		resp.end(indexTmpl[2]);
		yaku.disconnect();
	});
	yaku.on('error', function (err) {
		console.error('page', page + ':', err);
		resp.end();
		yaku.disconnect();
	});
	return true;
});

function render_404(resp) {
	resp.send(404, httpHeaders);
	resp.end(notFoundHtml);
}

function redirect_thread(resp, num, op) {
	resp.redirect(op + '#' + num, 302);
}

server.get('/:op', function (req, resp) {
	var num = parseInt(req.param('op'));
	if (!num)
		return req.next();
	var op = db.OPs[num];
	if (typeof op == 'undefined')
		return req.next();
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
		resp.write('<hr>\n');
	});
	write_thread_html(reader, resp, true);
	reader.on('end', function () {
		resp.write('[<a href=".">Return</a>]');
		resp.end(indexTmpl[2]);
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

function on_client (socket, retry) {
	if (socket.connection) {
		var client = new Okyaku;
		client.init(socket);
		clients[client.id] = client;
		console.log(client.id + " has IP " + client.ip);
	}
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

OK.init = function (socket) {
	this.ip = socket.connection.remoteAddress;
	this.id = socket.sessionId;
	this.socket = socket;
	this.watching = {};
	this.db = new db.Yakusoku;
	this.skipped = 0;
	socket.on('message', this.on_message.bind(this));
	socket.on('disconnect', this.on_disconnect.bind(this));
	socket.on('error', console.error.bind(console, 'socket:'));
	this.db.on('error', console.error.bind(console, 'redis:'));
};

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
