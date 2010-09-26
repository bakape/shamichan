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
		for (var j = 0; j < thread.length; j++)
			response.write(common.gen_post_html(thread[j]));
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

localClient.subscribe('/post/new', function (msg) {
	var num = post_counter++;
	now = new Date();
	var parsed = common.parse_name(msg.name);
	var post = {
		name: parsed[0],
		time: [now.getHours(), now.getMinutes(), now.getSeconds()],
		num: num,
		body: msg.frag
	};
	if (parsed[1]) {
		var trip = tripcode.hash(parsed[1]);
		if (trip)
			post.trip = trip;
	}
	if (msg.op && posts[msg.op] && !posts[msg.op].op)
		post.op = msg.op;

	var announce = common.clone(post);
	localClient.publish('/post/ok/' + msg.id, announce);
	localClient.publish('/thread/new', announce);
	/* And save this for later */
	post.id = msg.id;
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
	var post = posts[msg.num];
	if (post && post.id == msg.id) {
		localClient.publish('/frag', {num: msg.num, frag: msg.frag});
		post.body += msg.frag;
	}
});

localClient.subscribe('/post/done', function (msg) {
	var post = posts[msg.num];
	if (post && post.id == msg.id) {
		localClient.publish('/thread/done', {num: msg.num});
		delete post.id;
	}
});

bayeux.attach(server);
server.listen(8000);
