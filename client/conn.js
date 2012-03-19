(function () {

var socket, attempts, attemptTimer;

window.send = function (msg) {
	msg = JSON.stringify(msg);
	if (DEBUG)
		console.log('<', msg);
	socket.send(msg);
};

function on_message(e) {
	if (DEBUG)
		console.log('>', e.data);
	var msgs = JSON.parse(e.data);

	with_dom(function () {

	for (var i = 0; i < msgs.length; i++) {
		var msg = msgs[i];
		var op = msg.shift();
		var type = msg.shift();
		if (is_pubsub(type))
			syncs[op]++;
		dispatcher[type](msg, op);
	}

	});
}

function sync_status(msg, hover) {
	$('#sync').text(msg).attr('class', hover ? 'error' : '');
}

connSM.act('load', {start: 'conn'}).on('conn', function () {
	sync_status('Connecting...', false);
	attempts = 0;
	connect();
});

function connect() {
	socket = new SockJS(SOCKET_PATH);
	socket.onopen = connSM.feeder('open');
	socket.onclose = connSM.feeder('close');
	socket.onmessage = on_message;
}

connSM.act('conn', {open: 'syncing'}).act('reconn', {open: 'syncing'});
connSM.on('syncing', function () {
	sync_status('Syncing...', false);
	sessionId = Math.floor(Math.random() * 1e16) + 1;
	send([SYNCHRONIZE, sessionId, BOARD, syncs, BUMP, document.cookie]);
});

connSM.act('syncing', {sync: 'synced'}).on('synced', function () {
	sync_status('Synced.', false);
	attemptTimer = setTimeout(function () {
		attempts = 0;
	}, 10000);
});

connSM.wild('close', 'dropped');
connSM.on('dropped', function (e) {
	if (DEBUG)
		console.error('E:', e);
	if (attemptTimer) {
		clearTimeout(attemptTimer);
		attemptTimer = 0;
	}
	sync_status('Dropped.', true);
	if (attempts++ < 10)
		setTimeout(connSM.feeder('retry'), 250 * Math.pow(2,attempts));
});

connSM.act('dropped', {retry: 'reconn'}).on('reconn', function () {
	sync_status('Dropped. Reconnecting...', true);
	connect();
});

connSM.wild('invalid', 'out').act('out', {close: 'out'});
connSM.on('out', function (msg) {
	msg = (msg && msg[0]) ? 'Out of sync: ' + msg[0] : 'Out of sync.';
	sync_status(msg, true);
	if (attemptTimer) {
		clearTimeout(attemptTimer);
		attemptTimer = 0;
	}
	delete socket.onclose;
	socket.close();
	socket = null;
});

$(connSM.feeder('start'));

})();
