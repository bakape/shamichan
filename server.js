var http = require('http'),
    faye = require('./faye/faye-node');

var bayeux = new faye.NodeAdapter({ mount: '/msg', timeout: 45 });
var localClient = bayeux.getClient();

var server = http.createServer(function(request, response) {
	response.writeHead(200, {'Content-Type': 'text/plain'});
	response.end('Hello, non-Bayeux request');
});

var posts = {};
var post_counter = 2;

localClient.subscribe('/post/new', function (msg) {
	var num = post_counter++;
	now = new Date();
	var post = {
		name: msg.name.trim() || 'Anonymous',
		trip: '!!test',
		time: [now.getHours(), now.getMinutes(), now.getSeconds()],
		num: num,
		body: msg.frag
	};
	localClient.publish('/post/ok/' + msg.id, post);
	localClient.publish('/thread/new', post);
	post.id = msg.id;
	posts[num] = post;
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
		post.id = null;
	}
});

bayeux.attach(server);
server.listen(8000);
