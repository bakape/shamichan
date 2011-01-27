var common = require('./common'),
	config = require('./config'),
	fs = require('fs'),
	io = require('socket.io'),
	http = require('http'),
	pix = require('./pix'),
	db = require('./db'),
	Template = require('./lib/json-template').Template,
	tripcode = require('./tripcode'),
	util = require('util');

var clients = {};
var dispatcher = {};

function multisend(client, msgs) {
	client.socket.send(JSON.stringify(msgs));
}

function broadcast(msg, origin) {
	var thread_num = post.op || post.num;
	++syncNumber;
	msg = JSON.stringify(msg);
	var payload = '[' + msg + ']';
	for (id in clients) {
		var client = clients[id];
		if (!client.synced)
			continue;
		if (client.watching && client.watching != thread_num) {
			/* Client isn't in this thread so let them fall
			 * out of sync until something relevant comes up */
			client.defer_sync = syncNumber;
			continue;
		}
		if (id == origin) {
			/* Client won't increment SYNC since they won't
			 * receive the broadcasted message, so do manually */
			multisend(client, [[common.SYNCHRONIZE, syncNumber]]);
		}
		else if (client.defer_sync) {
			/* First catch them up, then send the new message */
			client.socket.send('[[' + common.SYNCHRONIZE + ',' +
					client.defer_sync + '],' + msg + ']');
		}
		else {
			/* Client is already in sync */
			client.socket.send(payload);
		}
		/* At this point the client must be caught up */
		client.defer_sync = null;
	}
	var now = new Date().getTime();
	backlog.push({when: now, msg: msg, thread: thread_num});
	cleanup_backlog(now);
}

function cleanup_backlog(now) {
	var limit = now - config.BACKLOG_PERIOD;
	/* binary search would be nice */
	while (backlog.length && backlog[0].when < limit) {
		backlog.shift();
		backlogLastDropped++;
	}
}

dispatcher[common.SYNCHRONIZE] = function (msg, client) {
	if (msg.length != 2)
		return false;
	var sync = msg[0], watching = msg[1];
	if (sync.constructor != Number)
		return false;
	if (watching) {
		var post = posts[watching];
		if (post && !post.op)
			client.watching = watching;
		else
			return false;
	}
	if (sync == syncNumber) {
		multisend(client, [[common.SYNCHRONIZE, syncNumber]]);
		client.synced = true;
		return true; /* already synchronized */
	}
	if (sync > syncNumber)
		return false; /* client in the future? */
	if (sync < backlogLastDropped)
		return false; /* client took too long */
	var logs = [];
	for (var i = sync - backlogLastDropped; i < backlog.length; i++) {
		var log = backlog[i];
		if (!watching || log.thread == watching)
			logs.push(log.msg);
	}
	logs.push('[' + common.SYNCHRONIZE + ',' + syncNumber + ']');
	client.socket.send('[' + logs.join() + ']');
	client.synced = true;
	return true;
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
oneeSama.image_view = pix.get_image_view;
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
	reader.on('post', function (post, has_next) {
		oneeSama.full = full_thread;
		response.write(oneeSama.mono(post));
		if (!has_next)
			response.write('</section><hr>\n');
	});
}

var indexTmpl = Template(fs.readFileSync('index.html', 'UTF-8'),
		{meta: '{{}}'}).expand(config).split(/\$[A-Z]+/);
var notFoundHtml = fs.readFileSync('../www/404.html');

function image_status(status) {
	multisend(this.client, [[common.IMAGE_STATUS, status]]);
}

function set_post_image(post, image, imgnm) {
	post.image = image;
	post.imgnm = imgnm;
	if (post.op)
		posts[post.op].thread.image_count++;
}

var httpHeaders = {'Content-Type': 'text/html; charset=UTF-8',
		'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT, -1',
		'Cache-Control': 'no-cache'};
var server = http.createServer(function(req, resp) {
	if (req.method.toLowerCase() == 'post') {
		var upload = new pix.ImageUpload(clients, allocate_post,
				set_post_image, announce_image, image_status);
		upload.handle_request(req, resp);
		return;
	}
	if (req.url == '/' && render_index(req, resp))
		return;
	m = req.url.match(/^\/(\d+)$/);
	if (m && render_thread(req, resp, m[1]))
		return;
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
		resp.write(indexTmpl[1]);
		resp.write('0' /* XXX sync */);
		resp.end(indexTmpl[2]);
		yaku.disconnect();
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
	reader.get_thread(parseInt(num));
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
		resp.write(indexTmpl[1]);
		resp.write('0' /* XXX sync */);
		resp.end(indexTmpl[2]);
		yaku.disconnect();
	});
	reader.on('error', function (err) {
		console.error('thread '+num+':', err);
		resp.end();
		yaku.disconnect();
	});
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
			bad_client(client, "Bad protocol.");
		}
	});
	socket.on('disconnect', function () {
		delete clients[id];
		finish_post_by(client);
		client.synced = false;
		client.db.quit();
	});
	socket.on('error', console.error.bind(console, 'socket:'));
	client.db.on('error', console.error.bind(console, 'redis:'));
}

function bad_client(client, msg) {
	console.error('Bad ' + client.ip + ': ' + msg);
	multisend(client, [[common.INVALID, msg]]);
	client.synced = false;
}

function valid_links(frag, state) {
	var links = {};
	var onee = new common.OneeSama(function (num, e) {
		var post = posts[num];
		if (post)
			links[num] = post.op || post.num;
	});
	onee.callback = function (frag) {};
	onee.state = state;
	onee.fragment(frag);
	return links;
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
	if (config.IMAGE_UPLOAD && !msg.op)
		return false;
	if (client.post)
		return true; /* image upload/fragment typing race */
	var frag = msg.frag;
	if (!frag || frag.match(/^\s*$/g))
		return false;
	return allocate_post(msg, null, null, client, function (err, alloc) {
		if (err) {
			/* TODO: Report */
			console.log(err);
			return;
		}
		multisend(client, [[common.ALLOCATE_POST, alloc]]);
	});
}

function allocate_post(msg, image, imgnm, client, callback) {
	if (!msg || typeof msg != 'object')
		return false;
	var post = {time: new Date().getTime()};
	var body = '';
	if (msg.frag !== undefined) {
		if (typeof msg.frag != 'string' || msg.frag.match(/^\s*$/g)
				|| msg.frag.length > common.MAX_POST_CHARS)
			return false;
		body = msg.frag;
	}
	if (typeof msg.op == 'number' && posts[msg.op] && !posts[msg.op].op)
		post.op = msg.op;
	if (client.watching && client.watching != post.op)
		return false;
	if (typeof msg.name != 'string')
		return false;
	var parsed = common.parse_name(msg.name);
	post.name = parsed[0];
	if (parsed[1] || parsed[2]) {
		var trip = tripcode.hash(parsed[1], parsed[2]);
		if (trip)
			post.trip = trip;
	}
	if (typeof msg.email == 'string')
		post.email = msg.email.trim().substr(0, 320);
	if (post.email == 'noko')
		delete post.email;
	if (image) {
		post.image = image;
		post.imgnm = imgnm;
	}
	/* XXX: What about the parse state?! */
	client.db.insert_post(post, body, client.ip, function (err, num) {
		if (err) {
			callback(err, null);
			return;
		}
		post.num = num;
		allocation_ok(post, client, callback);
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
	if (post.image)
		view.image = pix.get_image_view(post.image, post.imgnm,
				post.op);
	return view;
}

function allocation_ok(post, client, callback) {
	if (client.post) {
		/* Race condition... discard this */
		return callback('Already have a post.', null);
	}
	client.post = post.num;
	//posts[post.num] = post; // XXX
	post.state = [0, 0];
	post.links = valid_links(post.body, post.state);

	var view = get_post_view(post);
	broadcast([common.INSERT_POST, view], client.id);
	callback(null, view);
	if (!post.op) {
		/* New thread */
		post.thread = {image_count: 0, replies: [],
				last_bump: post.num, op: post};
		threads.unshift(post.thread);
	}
	else {
		var thread = posts[post.op].thread;
		thread.replies.push(post);
		if (post.image)
			thread.image_count++;
 		if (post.email != 'sage') {
			thread.last_bump = post.num;
			/* Bump thread */
			for (var i = 0; i < threads.length; i++) {
				if (threads[i] == thread) {
					threads.splice(i, 1);
					threads.unshift(thread);
					break;
				}
			}
		}
	}
}

function announce_image(info, client) {
	var post = client.post;
	broadcast([common.INSERT_IMAGE, post.num, info], client.id);
}

dispatcher[common.UPDATE_POST] = function (frag, client) {
	if (!frag || frag.constructor != String)
		return false;
	var post = client.post;
	if (!post)
		return false;
	var limit = common.MAX_POST_CHARS;
	if (frag.length > limit)
		return false;
	/* imporant: broadcast prior state */
	client.db.append_post(post, client.replying, frag, limit,
			function (err) {
		if (err)
			return bad_client(client, "Couldn't add text.");
		/* Okay, client should just cache the post */
	});
	/*
	var msg = [common.UPDATE_POST, post.num, frag].concat(post.state);
	var links = valid_links(frag, post.state);
	if (!isEmpty(links))
		msg.push({links: links});
	broadcast(msg, client.id);
	post.body += frag;
	for (var k in links)
		post.links[k] = links[k];
	*/
	return true;
}

function finish_post_by(client) {
	/* TODO: Should we check client.uploading? */
	if (!client.post)
		return false;
	client.db.finish_post(client.post, client.replying, function (err) {
		if (err)
			return bad_client(client, err);
		broadcast([common.FINISH_POST, post_id], client.id);
	});
	delete client.post;
	delete client.replying;
	client.editing = false;
	return true;
}

dispatcher[common.FINISH_POST] = function (msg, client) {
	if (msg.length)
		return false;
	return finish_post_by(client);
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

start_server();
