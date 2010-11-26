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

var threads = [];
var posts = {};
var clients = {};
var dispatcher = {};

var sync_number = 0;
var backlog = [];
var backlog_last_dropped = 0;

function multisend(client, msgs) {
	client.socket.send(JSON.stringify(msgs));
}

function broadcast(msg, post, origin) {
	var thread_num = post.op || post.num;
	++sync_number;
	msg = JSON.stringify(msg);
	var payload = '[' + msg + ']';
	for (id in clients) {
		var client = clients[id];
		if (!client.synced)
			continue;
		if (client.watching && client.watching != thread_num) {
			/* Client isn't in this thread so let them fall
			 * out of sync until something relevant comes up */
			client.defer_sync = sync_number;
			continue;
		}
		if (id == origin) {
			/* Client won't increment SYNC since they won't
			 * receive the broadcasted message, so do manually */
			multisend(client, [[common.SYNCHRONIZE, sync_number]]);
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
		backlog_last_dropped++;
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
	if (sync == sync_number) {
		multisend(client, [[common.SYNCHRONIZE, sync_number]]);
		client.synced = true;
		return true; /* already synchronized */
	}
	if (sync > sync_number)
		return false; /* client in the future? */
	if (sync < backlog_last_dropped)
		return false; /* client took too long */
	var logs = [];
	for (var i = sync - backlog_last_dropped; i < backlog.length; i++) {
		var log = backlog[i];
		if (!watching || log.thread == watching)
			logs.push(log.msg);
	}
	logs.push('[' + common.SYNCHRONIZE + ',' + sync_number + ']');
	client.socket.send('[' + logs.join() + ']');
	client.synced = true;
	return true;
}

post_env = {format_link: function (num, env) {
	var post = posts[num];
	if (post)
		env.callback(common.safe('<a href="'
				+ common.post_url(post, false)
				+ '">&gt;&gt;' + num + '</a>'));
	else
		env.callback('>>' + num);
	},
	image_view: pix.get_image_view,
	dirs: {src_url: config.IMAGE_URL, thumb_url: config.THUMB_URL}
};

function write_thread_html(thread, response) {
	var first = common.gen_thread(thread[0], post_env);
	var ending = first.pop();
	response.write(first.join(''));
	for (var i = 1; i < thread.length; i++)
		response.write(common.gen_post_html(thread[i], post_env));
	response.write(ending + '<hr>\n');
}

var index_tmpl = Template(fs.readFileSync('index.html', 'UTF-8'),
		{meta: '{{}}'}).expand(config).split(/\$[A-Z]+/);
var notfound_html = fs.readFileSync('../www/404.html');

var http_headers = {'Content-Type': 'text/html; charset=UTF-8',
		'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT, -1',
		'Cache-Control': 'no-cache'};
var server = http.createServer(function(req, resp) {
	if (req.method.toLowerCase() == 'post') {
		var upload = new pix.ImageUpload(clients, allocate_post,
			announce_image);
		upload.handle_request(req, resp);
		return;
	}
	if (req.url == '/' && render_index(req, resp))
		return;
	m = req.url.match(/^\/(\d+)$/);
	if (m && render_thread(req, resp, m[1]))
		return;
	resp.writeHead(404, http_headers);
	resp.end(notfound_html);
});

function render_index(req, resp) {
	resp.writeHead(200, http_headers);
	resp.write(index_tmpl[0]);
	for (var i = 0; i < threads.length; i++)
		write_thread_html(threads[i], resp);
	resp.write(index_tmpl[1]);
	resp.write(sync_number.toString());
	resp.end(index_tmpl[2]);
	return true;
}

function render_thread(req, resp, num) {
	var post = posts[parseInt(num)];
	if (!post)
		return false;
	if (post.op) {
		resp.writeHead(301, {Location: '/'+post.op+'#q'+post.num});
		resp.end();
		return true;
	}
	resp.writeHead(200, http_headers);
	resp.write(index_tmpl[0]);
	write_thread_html(post.thread, resp);
	resp.write('[<a href=".">Return</a>]');
	resp.write(index_tmpl[1]);
	resp.write(sync_number.toString());
	resp.end(index_tmpl[2]);
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
			watching: null, ip: ip};
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
			console.log("Got invalid message " + data);
			multisend(client, [[common.INVALID]]);
			client.synced = false;
		}
	});
	socket.on('disconnect', function () {
		delete clients[id];
		if (client.post)
			finish_post(client.post, id);
		client.synced = false;
	});
	socket.on('error', function (err) {
		console.log(err);
	});
}

function valid_links(frag, state) {
	var links = {};
	env = {callback: function (frag) {}, format_link: function (num, e) {
		var post = posts[num];
		if (post)
			links[num] = post.op || post.num;
	}};
	common.format_fragment(frag, state, env);
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
	if (!frag || typeof frag != 'string' || frag.match(/^\s*$/g))
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
	if (!msg)
		return false;
	if (msg.frag && msg.frag.length > common.MAX_POST_CHARS)
		return false;
	var post = {
		time: new Date().getTime(),
		editing: true,
		body: msg.frag || '',
	};
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
	db.insert_post(post, client.ip, function (err, num) {
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
	posts[post.num] = post;
	var state = common.initial_post_state();
	post.links = valid_links(post.body, state);
	var view = get_post_view(post);
	broadcast([common.INSERT_POST, view], view, client.id);
	callback(null, view);
	/* Store some extra state for later */
	post.state = state;
	client.post = post;
	if (!post.op) {
		/* New thread */
		post.thread = [post];
		threads.unshift(post.thread);
	}
	else {
		var thread = posts[post.op].thread;
		thread.push(post);
 		if (post.email != 'sage') {
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
	broadcast([common.INSERT_IMAGE, post.num, info], post, client.id);
}

dispatcher[common.UPDATE_POST] = function (frag, client) {
	if (!frag || frag.constructor != String)
		return false;
	var post = client.post;
	if (!post || !post.editing)
		return false;
	if (post.body.length + frag.length > common.MAX_POST_CHARS)
		return false;
	/* imporant: broadcast prior state */
	var msg = [common.UPDATE_POST, post.num, frag].concat(post.state);
	var links = valid_links(frag, post.state);
	if (!isEmpty(links))
		msg.push({links: links});
	broadcast(msg, post, client.id);
	post.body += frag;
	for (var k in links)
		post.links[k] = links[k];
	return true;
}

function finish_post(post, owner_id) {
	/* TODO: Should we check client.uploading? */
	broadcast([common.FINISH_POST, post.num], post, owner_id);
	post.editing = false;
	delete post.state;
	db.update_post(post.num, post.body, function (ok) {
		if (!ok) {
			/* TODO */
			console.log("Couldn't save final post #" + post.num);
		}
	});
}

dispatcher[common.FINISH_POST] = function (msg, client) {
	if (msg.length)
		return false;
	var post = client.post;
	if (!post || !post.editing)
		return false;
	finish_post(post, client.id);
	client.post = null;
	return true;
}

function populate_threads(parents, callback) {
	db.get_posts(false, function (err, post) {
		if (err) throw err;
		if (post) {
			posts[post.num] = post;
			parents[post.op].thread.push(post);
			if (post.email != 'sage')
				parents[post.op].last = post.num;
		}
		else {
			var ts = [];
			for (var num in parents)
				ts.push(parents[num]);
			ts.sort(function (a, b) { return a.last < b.last; });
			for (var i = 0; i < ts.length; i++)
				ts[i] = ts[i].thread;
			threads = ts;
			callback();
		}
	});
}

function load_threads(callback) {
	var parents = {};
	db.get_posts(true, function (err, post) {
		if (err) throw err;
		if (post) {
			var thread = [post];
			post.thread = thread;
			posts[post.num] = post;
			parents[post.num] = {thread: thread, last: post.num};
		}
		else
			populate_threads(parents, callback);
	});
}

function start_server() {
	server.listen(config.PORT);
	var listener = io.listen(server, {
		transports: ['websocket', 'server-events', 'htmlfile',
			'xhr-multipart', 'xhr-polling']
	});
	listener.on('connection', on_client);
	listener.on('error', function (err) {
		console.log(err);
	});
}

db.check_tables(function () {
	console.log("Database OK.");
	load_threads(start_server);
});
