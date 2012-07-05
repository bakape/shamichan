var _ = require('../lib/underscore'),
    caps = require('./caps'),
    config = require('../config'),
    db = require('../db'),
    web = require('./web');

var RES = require('./state').resources;

web.route_get_auth(/^\/admin$/, function (req, resp) {
	if (!caps.is_admin_ident(req.ident))
		return web.render_404(resp);
	var board = req.board || 'moe';
	if (!caps.can_access_board(req.ident, board))
		return web.render_404(resp);

	var img = _.template('<a href="' + board + '/{{num}}">'
			+ '<img alt="{{num}}" title="Thread {{num}}" src="'
			+ config.MEDIA_URL + 'thumb/{{thumb}}" width=50 '
			+ 'height=50></a>\n');
	var limit = parseInt(req.query.limit, 10) || 0;
	var ctr = 0;

	resp.writeHead(200);
	resp.write(RES.filterTmpl[0]);
	resp.write('<h2>Limit ' + limit + '</h2>\n');

	var filter = new db.Filter(board);
	filter.get_all(limit);

	filter.on('thread', function (thread) {
		resp.write(img(thread));
		ctr += 1;
	});
	filter.once('end', function () {
		resp.write('<br>' + ctr + ' thread(s).');
		resp.end(RES.filterTmpl[1]);
	});
	filter.once('error', function (err) {
		resp.end('<br><br>Error: ' + escape(err));
	});
});

web.route_post_auth(/^\/admin$/, function (req, resp) {
	if (!caps.is_admin_ident(req.ident))
		return web.render_404(resp);

	var threads = req.body.threads.split(',').map(function (x) {
		return parseInt(x, 10);
	}).filter(function (x) {
		return !isNaN(x);
	});

	var yaku = new db.Yakusoku(null);
	yaku.remove_posts(threads, function (err, dels) {

		// XXX: Can't disconnect right away.
		//      Does its business in the background.
		//      Grrr. Hack for now.
		setTimeout(function () {
			yaku.disconnect();
		}, 30 * 1000);

		if (err) {
			web.dump_server_error(resp, err);
			return;
		}
		resp.writeHead(200, web.noCacheHeaders);
		resp.end();
	});
});
