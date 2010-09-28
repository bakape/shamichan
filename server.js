var common = require('./common'),
	faye = require('./faye/faye-node'),
	fs = require('fs'),
	http = require('http'),
	tripcode = require('./tripcode');

var bayeux = new faye.NodeAdapter({ mount: '/msg', timeout: 45 });
var localClient = bayeux.getClient();

var threads = [];
var posts = {};
var post_counter = 1;

function write_threads_html(response) {
	for (var i = 0; i < threads.length; i++) {
		var thread = threads[i];
		response.write('\t<ul name="thread' + thread[0].num + '">\n');
		for (var j = 0; j < thread.length; j++) {
			var post = thread[j];
			response.write(common.gen_post_html(post));
		}
		response.write('\t</ul>\n');
	}
}

var index_tmpl = fs.readFileSync('index.html', 'UTF-8').split("$THREADS");

var server = http.createServer(function(request, response) {
	response.writeHead(200, {'Content-Type': 'text/html; charset=UTF-8'});
	response.write(index_tmpl[0]);
	write_threads_html(response);
	response.end(index_tmpl[1]);
});

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

function multiple_newlines(frag) {
	return frag.split('\n', 3).length > 2;
}

localClient.subscribe('/post/new', function (msg) {
	if (!validate(msg, {name: String, frag: String, id: Number}))
		return;
	if (!msg.frag.replace(/[ \n]/g, '') || multiple_newlines(msg.frag))
		return;
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

	var announce = common.clone(post);
	localClient.publish('/post/ok/' + msg.id, announce);
	localClient.publish('/thread/new', announce);
	/* And save this for later */
	post.id = msg.id;
	post.state = common.initial_post_state();
	common.format_fragment(post.body, post.state, null);
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
});

localClient.subscribe('/post/frag', function (msg) {
	if (!validate(msg, {frag: String, num: Number, id: Number}))
		return;
	if (multiple_newlines(msg.frag))
		return;
	var post = posts[msg.num];
	if (post && post.editing && post.id == msg.id) {
		var announce = [post.state[0], post.state[1]]; /* TEMP */
		localClient.publish('/frag',
			{num: msg.num, frag: msg.frag, state: announce});
		post.body += msg.frag;
		/* update state */
		common.format_fragment(msg.frag, post.state, null);
	}
});

localClient.subscribe('/post/done', function (msg) {
	if (!validate(msg, {num: Number, id: Number}))
		return;
	var post = posts[msg.num];
	if (post && post.editing && post.id == msg.id) {
		localClient.publish('/thread/done', {num: msg.num});
		post.editing = false;
		delete post.id;
		delete post.state;
	}
});

bayeux.attach(server);
server.listen(8000);
