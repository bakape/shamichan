var common = require('./common'),
	config = require('./config').config,
	exec = require('child_process').exec,
	formidable = require('../formidable'),
	fs = require('fs'),
	io = require('../socket.io'),
	jsontemplate = require('./json-template'),
	http = require('http'),
	path = require('path'),
	tripcode = require('./tripcode');

var threads = [];
var posts = {};
var post_counter = 1;
var clients = {};
var dispatcher = {};

var sync_number = 0;
var backlog = [];
var backlog_last_dropped = 0;
var BL_WHEN = 0, BL_MSG = 1, BL_THREAD = 2;

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
		if (client.watching && client.watching != thread_num)
			multisend(client, [[common.INVALID]]);
		else if (id == origin) {
			/* Client won't increment SYNC since they won't
			 * receive the broadcasted message, so do manually */
			multisend(client, [[common.SYNCHRONIZE, sync_number]]);
		}
		else if (client.synced)
			client.socket.send(payload);
	}
	var now = new Date().getTime();
	backlog.push([now, msg, thread_num]);
	cleanup_backlog(now);
}

function cleanup_backlog(now) {
	var limit = now - config.BACKLOG_PERIOD;
	/* binary search would be nice */
	while (backlog.length && backlog[0][BL_WHEN] < limit) {
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
		if (!watching || log[BL_THREAD] == watching)
			logs.push(log[BL_MSG]);
	}
	logs.push('[' + common.SYNCHRONIZE + ',' + sync_number + ']');
	client.socket.send('[' + logs.join() + ']');
	client.synced = true;
	return true;
}

post_env = {format_link: function (num, env) {
	var post = posts[num];
	if (post)
		env.callback(common.safe('<a href="' + common.post_url(post)
				+ '">&gt;&gt;' + num + '</a>'));
	else
		env.callback('>>' + num);
}};

function write_thread_html(thread, response) {
	response.write('<section id="thread' + thread[0].num + '">\n');
	for (var i = 0; i < thread.length; i++)
		response.write(common.gen_post_html(thread[i], post_env));
	response.write('</section>\n');
}

var index_tmpl = jsontemplate.Template(fs.readFileSync('index.html', 'UTF-8')
		).expand(config).split(/\$[A-Z]+/);
var notfound_html = fs.readFileSync('www/404.html');

var http_headers = {'Content-Type': 'text/html; charset=UTF-8'};
var server = http.createServer(function(req, resp) {
	if (req.method.toLowerCase() == 'post') {
		handle_upload(req, resp);
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
	resp.write(sync_number.toString());
	resp.write(index_tmpl[1]);
	for (var i = 0; i < threads.length; i++)
		write_thread_html(threads[i], resp);
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
	resp.write(sync_number.toString());
	resp.write(index_tmpl[1]);
	write_thread_html(post.thread, resp);
	resp.end(index_tmpl[2]);
	return true;
}

function client_call(resp, func, param) {
	param = param ? JSON.stringify(param) : '';
	resp.writeHead(200, {'Content-Type': 'text/html; charset=UTF-8'});
	resp.end('<!doctype html>\n<title></title>\n<script>'
		+ 'parent.' + func + '(' + param + ');</script>');
}

function handle_upload(req, resp) {
	var form = new formidable.IncomingForm();
	form.maxFieldsSize = 512;
	form.onPart = function (part) {
		if (part.filename && part.name == 'image')
			form.handlePart(part);
		else if (!part.filename && part.name == 'client_id')
			form.handlePart(part);
		else {
			this._error('Superfluous field.');
			if (part.path)
				console.log(part.path);
		}
	};
	form.parse(req, function (err, fields, files) {
		if (err) {
			console.log("Upload error: " + err);
			var code = 500;
			err = '' + (err.message || err);
			return client_call(resp, 'upload_error', err);
		}
		var image = files.image;
		if (!image)
			return client_call(resp, 'upload_error',
					'No image supplied.');
		var fail = function(text) {
			fs.unlink(image.path);
			client_call(resp, 'upload_error', text);
		}
		image.ext = path.extname(image.filename);
		if (config.IMAGE_EXTS.indexOf(image.ext.toLowerCase()) < 0)
			return fail('Invalid image format.');
		image.client_id = parseInt(fields.client_id);
		if (!(image.client_id in clients))
			return fail('Invalid client id.');
		image.resp = resp;
		resize_image(image, upload_image);
	});
}

function resize_image(image, callback) {
	image.thumb = image.path + '_thumb';
	var path = image.ext.replace('.', '') + ':' + image.path;
	exec('convert ' + path + ' -gamma 0.454545 -filter lanczos -resize '
		+ config.THUMB_DIMENSIONS + ' -gamma 2.2 -quality '
		+ config.THUMB_QUALITY + ' ' + image.thumb,
		image_handler(image, 'Conversion error.', upload_image));
}

function image_handler(image, err_desc, callback) {
	return function (error, stdout, stderr) {
		if (error != null)
			console.log(error);
		if (stdout)
			console.log(stdout);
		if (stderr)
			console.log(stderr);
		if (error == null && !stderr)
			callback(image);
		else {
			client_call(image.resp, 'upload_error', err_desc);
			fs.unlink(image.path);
			fs.unlink(image.thumb);
		}
	};
}

function upload_image(image) {
	var base = new Date().getTime();
	var dest = path.join(config.IMAGE_DIR, base + image.ext);
	var thumb_dest = path.join(config.THUMB_DIR, base + '.jpg');
	exec('mv -- ' + image.path + ' ' + dest, image_handler(image,
	"Couldn't publish image.", function (image) {
	exec('mv -- ' + image.thumb + ' ' + thumb_dest, image_handler(image,
	"Couldn't publish thumbnail.", function (image) {
		console.log(image.client_id + ' successfully uploaded ' + dest);
		var dest_url = config.IMAGE_URL + base + image.ext;
		var thumb_url = config.THUMB_URL + base + '.jpg';
		client_call(image.resp, 'upload_complete', {
			src: dest_url, thumb: thumb_url,
			filename: image.filename,
		});
	}));
	}));
}

function on_client (socket) {
	var id = socket.sessionId;
	var client = {id: id, socket: socket, post: null, synced: false,
			watching: null};
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
	msg = msg.shift();
	if (!validate(msg, {name: String, frag: String}))
		return false;
	if (!msg.frag.replace(/[ \n]/g, ''))
		return false;
	var post = {
		time: new Date().getTime(),
		editing: true,
		body: msg.frag
	};
	if (is_integer(msg.op) && posts[msg.op] && !posts[msg.op].op)
		post.op = msg.op;
	if (client.watching && client.watching != post.op)
		return false;
	var parsed = common.parse_name(msg.name);
	post.name = parsed[0];
	if (parsed[1] || parsed[2]) {
		var trip = tripcode.hash(parsed[1], parsed[2]);
		if (trip)
			post.trip = trip;
	}
	if (msg.email && msg.email.constructor == String)
		post.email = msg.email.trim().substr(0, 320);

	/* No going back now */
	post.num = post_counter++;
	posts[post.num] = post;
	var state = common.initial_post_state();
	var links = valid_links(post.body, state);
	if (!isEmpty(links))
		post.links = links;
	broadcast([common.INSERT_POST, post], post, client.id);
	multisend(client, [[common.ALLOCATE_POST, post]]);
	/* Store some extra state for later */
	post.links = links;
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
	return true;
}

dispatcher[common.UPDATE_POST] = function (frag, client) {
	if (!frag || frag.constructor != String)
		return false;
	var post = client.post;
	if (!post || !post.editing)
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
	broadcast([common.FINISH_POST, post.num], post, owner_id);
	post.editing = false;
	delete post.state;
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

server.listen(config.PORT);
var listener = io.listen(server, {
	transports: ['websocket', 'server-events', 'htmlfile', 'xhr-multipart',
		'xhr-polling']
});
listener.on('connection', on_client);
