var common = require('./common'),
	config = require('./config').config,
	fs = require('fs'),
	io = require('../socket.io'),
	jsontemplate = require('./json-template'),
	http = require('http'),
	tripcode = require('./tripcode');

var threads = [];
var posts = {};
var post_counter = 1;
var clients = {};
var dispatcher = {};

var sync_number = 0;
var backlog = [];
var backlog_last_dropped = 0;

function multisend(client, msgs) {
	client.socket.send(JSON.stringify(msgs));
}

function broadcast(msg, except) {
	msg = JSON.stringify(msg);
	var payload = '[' + msg + ']';
	for (id in clients) {
		var client = clients[id];
		if (id != except && client.synced)
			client.socket.send(payload);
	}
	var now = new Date().getTime();
	++sync_number;
	backlog.push([now, msg]);
	cleanup_backlog(now);
}

function cleanup_backlog(now) {
	var limit = now - config.BACKLOG_PERIOD;
	/* binary search would be nice */
	while (backlog.length && backlog[0][0] < limit) {
		backlog.shift();
		backlog_last_dropped++;
	}
}

dispatcher[common.SYNCHRONIZE] = function (msg, client) {
	if (msg.length != 1)
		return false;
	var sync = msg[0];
	if (sync.constructor != Number)
		return false;
	if (sync == sync_number) {
		multisend(client, [[common.SYNCHRONIZE]]);
		client.synced = true;
		return true; /* already synchronized */
	}
	if (sync > sync_number)
		return false; /* client in the future? */
	if (sync < backlog_last_dropped)
		return false; /* client took too long */
	var logs = [];
	for (var i = sync - backlog_last_dropped; i < backlog.length; i++)
		logs.push(backlog[i][1]);
	logs.push('[' + common.SYNCHRONIZE + ']');
	client.socket.send('[' + logs.join() + ']');
	client.synced = true;
	return true;
}

function write_threads_html(response) {
	for (var i = 0; i < threads.length; i++) {
		var thread = threads[i];
		response.write('\t<ul id="thread' + thread[0].num + '">\n');
		for (var j = 0; j < thread.length; j++) {
			var post = thread[j];
			response.write(common.gen_post_html(post));
		}
		response.write('\t</ul>\n');
	}
}

var index_tmpl = jsontemplate.Template(fs.readFileSync('index.html', 'UTF-8')
		).expand(config).split(/\$[A-Z]+/);

var server = http.createServer(function(request, response) {
	response.writeHead(200, {'Content-Type': 'text/html; charset=UTF-8'});
	response.write(index_tmpl[0]);
	response.write(sync_number.toString());
	response.write(index_tmpl[1]);
	write_threads_html(response);
	response.end(index_tmpl[2]);
});

function on_client (socket) {
	var id = socket.sessionId;
	var client = {id: id, socket: socket, post: null, synced: false};
	clients[id] = client;
	socket.on('message', function (data) {
		msg = JSON.parse(data);
		var type = common.INVALID;
		if (client.post && msg.constructor == String)
			type = common.UPDATE_POST;
		else if (msg.constructor == Array)
			type = msg.shift();
		var func = dispatcher[type];
		if (!func || !func(msg, client)) {
			console.log("Got invalid message " + data);
			multisend(client, [[common.INVALID]]);
		}
	});
	socket.on('disconnect', function () {
		delete clients[id];
		if (client.post)
			finish_post(client.post, id);
	});
}

function is_integer(n) {
	return (typeof(n) == 'number' && parseFloat(n) == parseInt(n)
			&& !isNaN(n));
}

function validate(msg, schema) {
	if (msg == null || typeof(msg) != 'object')
		return false;
	for (var k in schema) {
		var m = msg[k];
		if (m == null || m.constructor != schema[k])
			return false;
		if (schema[k] == Number && !is_integer(m))
			return false;
	}
	return true;
}

dispatcher[common.ALLOCATE_POST] = function (msg, client) {
	if (msg.length != 1)
		return false;
	msg = msg.shift();
	if (!validate(msg, {name: String, frag: String}))
		return false;
	if (!msg.frag.replace(/[ \n]/g, ''))
		return false;
	var num = post_counter++;
	now = new Date();
	var parsed = common.parse_name(msg.name);
	var post = {
		name: parsed[0],
		time: [now.getHours(), now.getMinutes(), now.getSeconds()],
		num: num,
		editing: true,
		body: msg.frag
	};
	if (parsed[1] || parsed[2]) {
		var trip = tripcode.hash(parsed[1], parsed[2]);
		if (trip)
			post.trip = trip;
	}
	if (is_integer(msg.op) && posts[msg.op] && !posts[msg.op].op)
		post.op = msg.op;

	multisend(client, [[common.ALLOCATE_POST, post]]);
	broadcast([common.INSERT_POST, post], client.id);
	/* And save this for later */
	post.state = common.initial_post_state();
	common.format_fragment(post.body, post.state, null);
	client.post = post;
	posts[num] = post;
	if (!post.op) {
		/* New thread */
		post.thread = [post];
		threads.unshift(post.thread);
	}
	else {
		var thread = posts[post.op].thread;
		thread.push(post);
		/* Bump thread */
		for (var i = 0; i < threads.length; i++) {
			if (threads[i] == thread) {
				threads.splice(i, 1);
				threads.unshift(thread);
				break;
			}
		}
	}
	return true;
}

dispatcher[common.UPDATE_POST] = function (frag, client) {
	if (!frag || frag.constructor != String)
		return false;
	var post = client.post;
	if (!post || !post.editing)
		return false;
	var msg = [common.UPDATE_POST, post.num, frag].concat(post.state);
	broadcast(msg, client.id);
	post.body += frag;
	common.format_fragment(frag, post.state, null); /* update state */
	return true;
}

function finish_post(post, owner_id) {
	broadcast([common.FINISH_POST, post.num], owner_id);
	post.editing = false;
	delete post.state;
}

dispatcher[common.FINISH_POST] = function (msg, client) {
	if (msg.length)
		return false;
	var post = client.post;
	if (!post.editing)
		return false;
	finish_post(post, client.id);
	client.post = null;
	return true;
}

server.listen(config.PORT);
var listener = io.listen(server, {
	transports: ['websocket', 'server-events', 'htmlfile', 'xhr-multipart',
		'xhr-polling']
});
listener.on('connection', on_client);
