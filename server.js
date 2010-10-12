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
		).expand(config).split('$THREADS');

var server = http.createServer(function(request, response) {
	response.writeHead(200, {'Content-Type': 'text/html; charset=UTF-8'});
	response.write(index_tmpl[0]);
	write_threads_html(response);
	response.end(index_tmpl[1]);
});

function on_client (socket) {
	var id = socket.sessionId;
	var client = {id: id, socket: socket, post: null};
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
			common.send(socket, [common.INVALID]);
			socket.close();
		}
	});
	socket.on('disconnect', function () {
		delete clients[id];
		if (client.post)
			finish_post(client.post, id);
	});
}

function broadcast(msg, except) {
	socket.broadcast(JSON.stringify(msg), except);
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
	if (parsed[1]) {
		var trip = tripcode.hash(parsed[1]);
		if (trip)
			post.trip = trip;
	}
	if (is_integer(msg.op) && posts[msg.op] && !posts[msg.op].op)
		post.op = msg.op;

	common.send(client.socket, [common.ALLOCATE_POST, post]);
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
var socket = io.listen(server, {
	transports: ['websocket', 'server-events', 'htmlfile', 'xhr-multipart',
		'xhr-polling']
});
socket.on('connection', on_client);
