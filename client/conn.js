(function () {

var socket;

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
		var type = msg.shift();
		/* Pub-sub messages have an extra OP-num entry */
		var op;
		if (is_pubsub(type)) {
			op = msg.pop();
			syncs[op]++;
		}
		dispatcher[type](msg, op);
	}

	});
}

connSM.act('load', {start: 'conn'}).on('conn', function () {
	sync_status('Connecting...', false);
	socket = new SockJS(SOCKET_PATH);
	socket.onopen = connSM.feeder('open');
	socket.onclose = connSM.feeder('close');
	socket.onmessage = on_message;
});

connSM.act('conn', {open: 'syncing'}).on('syncing', function () {
	sync_status('Syncing...', false);
	sessionId = Math.floor(Math.random() * 1e17) + 1;
	send([SYNCHRONIZE, sessionId, BOARD, syncs, BUMP, document.cookie]);
});

connSM.wild('close', 'dropped').on('dropped', function (e) {
	if (DEBUG)
		console.error('E:', e);
	sync_status('Dropped.', true);
});

connSM.wild('invalid', 'out').act('out', {close: 'out'});
connSM.on('out', function (msg) {
	msg = (msg && msg[0]) ? 'Out of sync: ' + msg[0] : 'Out of sync.';
	sync_status(msg, true);
	delete socket.onclose;
	socket.close();
	socket = null;
});

$(function () {
	connSM.feed('start');
});

})();
