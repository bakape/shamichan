/*
 * Handles the brunt of the post-related websocket calls
 */

var $ = require('jquery'),
	common = require('../common'),
	main = require('./main'),
	posts = require('./posts'),
	state = require('./state');

// The actual object of handler functions for websocket calls
var dispatcher = main.dispatcher;

dispatcher[common.INSERT_POST] = function(msg) {
	// TODO: msg[0] is legacy. Could use a fix after we shift in the new client
	msg = msg[1];
	const isThread = !msg.op;
	if (isThread)
		state.syncs[msg.num] = 1;
	msg.editing = true;

	// Did I create this post?
	var el;
	const msgNonce = msg.nonce;
	delete msg.nonce;
	const myNonce = posts.nonce.get_nonces()[msgNonce];
	var bump = state.page.get('live');
	if (myNonce && myNonce.tab === state.page.get('tabID')) {
		// posted in this tab; transform placeholder
		state.ownPosts[msg.num] = true;
		main.oneeSama.trigger('insertOwnPost', msg);
		main.postSM.feed('alloc', msg);
		bump = false;
		// delete only after a delay so all tabs notice that it's ours
		setTimeout(posts.nonce.destroy_nonce.bind(null, msgNonce), 10000);
		// if we've already made a placeholder for this post, use it
		if (main.postForm && main.postForm.el)
			el = main.postForm.el;
	}

	// Add to my post set
	if (myNonce) {
		msg.mine = true;
		state.mine.write(msg.num, state.mine.now());
	}

	// TODO: Shift the parrent sections replies on board pages

	// TODO: Bump thread to top, if not saging

	new posts[isThread ? 'Section' : 'Article']({
		// Create model
		model: new posts.models[isThread ? 'Thread' : 'Post'](msg),
		id: msg.num,
		el: el
	});
};

// Move thread to the archive board
dispatcher[common.MOVE_THREAD] = function(msg) {
	msg = msg[0];
	var model = new posts.ThreadModel(msg);
	main.oneeSama.links = msg.links;
	new posts.Section({
		model: model,
		id: msg.num
	});
};

dispatcher[common.INSERT_IMAGE] = function(msg) {
	var model = state.posts.get(msg[0]);
	// Did I just upload this?
	if (main.postModel && main.postModel.get('num') == msg[0]) {
		if (model)
			model.set('image', msg[1], {silent: true});
		main.postForm.insertUploaded(msg[1]);
	}
	else if (model)
		model.set('image', msg[1]);
};

dispatcher[common.UPDATE_POST] = function(msg) {
	const num = msg[0],
		links = msg[4],
		msgState = [msg[2] || 0, msg[3] || 0];
	var extra = msg[5],
		model = state.posts.get(num);

	// TODO: Add backlinks

	if (model) {
		model.set({
			body: model.get('body') + msg[1],
			state: msgState
		});
	}

	// Am I updating my own post?
	if (num in state.ownPosts) {
		if (extra)
			extra.links = links;
		else
			extra = {links: links};
		main.oneeSama.trigger('insertOwnPost', extra);
		return;
	}

	// TODO: Make this prettier
	var bq = $('#' + num + ' > blockquote');
	if (bq.length) {
		main.oneeSama.dice = extra && extra.dice;
		main.oneeSama.links = links || {};
		main.oneeSama.callback = inject;
		main.oneeSama.buffer = bq;
		main.oneeSama.state = msgState;
		main.oneeSama.fragment(msg[1]);
	}
};

// Add various additional tags inside the blockqoute
var inject = exports.inject = function(frag) {
	var $dest = this.buffer;
	for (var i = 0; i < this.state[1]; i++)
		$dest = $dest.children('del:last');
	if (this.state[0] == common.S_QUOTE)
		$dest = $dest.children('em:last');
	if (this.strong)
		$dest = $dest.children('strong:last');
	var out = null;
	if (frag.safe) {
		var m = frag.safe.match(/^<(\w+)>$/);
		if (m)
			out = document.createElement(m[1]);
		else if (/^<\/\w+>$/.test(frag.safe))
			out = '';
	}
	if (out === null) {
		if (Array.isArray(frag))
			out = $(common.flatten(frag).join(''));
		else
			out = common.escape_fragment(frag);
	}
	if (out)
		$dest.append(out);
	return out;
};

// Make the text spoilers toggle revealing on click
var touchable_spoiler_tag = exports.touchable_spoiler_tag = function(del) {
	del.html = '<del onclick="void(0)">';
};
main.oneeSama.hook('spoilerTag', touchable_spoiler_tag);

dispatcher[common.FINISH_POST] = function(msg) {
	const num = msg[0];
	delete state.ownPosts[num];
	var model = state.posts.get(num);
	if (model)
		model.set('editing', false);
};

dispatcher[common.DELETE_POSTS] = function(msg) {
	msg.forEach(function(num) {
		var model = state.posts.get(num);
		if (model)
			model.destroy();

		// TODO: Free up post focus, if any
	});
};

dispatcher[common.DELETE_THREAD] = function(msg, op) {
	delete state.syncs[op];
	delete state.ownPosts[op];

	if (main.postModel) {
		const num = main.postModel.get('num');
		if ((main.postModel.get('op') || num) === op)
			main.postSM.feed('done');
		if (num === op)
			return;
	}

	var model = state.getThread(op);
	if (model)
		model.destroy();
};

dispatcher[common.LOCK_THREAD] = function(msg, op) {
	var model = state.getThread(op);
	if (model)
		model.set('locked', true);
};

dispatcher[common.UNLOCK_THREAD] = function(msg, op) {
	var model = state.getThread(op);
	if (model)
		model.set('locked', false);
};

dispatcher[common.DELETE_IMAGES] = function(msg) {
	msg.forEach(function(num) {
		var model = state.posts.get(num);
		if (model)
			model.unset('image');
	});
};

dispatcher[common.SPOILER_IMAGES] = function(msg) {
	msg.forEach(function(info) {
		var model = state.posts.get(info[0]);
		if (model)
			model.trigger('spoiler',info[1]);
	});
};

dispatcher[common.SYNCHRONIZE] = main.connSM.feeder('sync');
dispatcher[common.INVALID] = main.connSM.feeder('invalid');

dispatcher[common.ONLINE_COUNT] = function(msg){
	$('#onlineCount').text('['+msg[0]+']');
};

// Sync settings to server
dispatcher[common.HOT_INJECTION] = function(msg){
	// Request new varibles, if hashes don't match
	if (msg[0] == false && msg[1] != state.configHash)
		main.send([common.HOT_INJECTION, true]);
	// Update variables and hash
	else if (msg[0] == true) {
		state.configHash = msg[1];
		state.hotConfig.set(msg[2]);
	}
};

/*
 * TODO: These are used only for the Admin panel. Would be nice, if we could
 * set those in admin/client.js. Would need to export main.js outside the bundle
 * then.
 */
dispatcher[common.MODEL_SET] = function (msg, op) {};
dispatcher[common.COLLECTION_RESET] = function (msg, op) {};
dispatcher[common.COLLECTION_ADD] = function (msg, op) {};

// Include other additional modules
require('./amusement');

// Connect to the server
require('./connection');
