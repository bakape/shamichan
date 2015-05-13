/*
 * Handles the brunt of the post-related websocket calls
 */
'use strict';

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
	const myNonce = posts.nonce.get()[msgNonce];
	var bump = state.page.get('live');
	if (myNonce && myNonce.tab === state.page.get('tabID')) {
		// posted in this tab; transform placeholder
		state.ownPosts[msg.num] = true;
		main.oneeSama.trigger('insertOwnPost', msg);
		main.postSM.feed('alloc', msg);
		bump = false;
		// delete only after a delay so all tabs notice that it's ours
		setTimeout(posts.nonce.destroy.bind(null, msgNonce), 10000);
		// if we've already made a placeholder for this post, use it
		if (main.postForm && main.postForm.el)
			el = main.postForm.el;
	}
	// Add to my post set
	if (myNonce) {
		msg.mine = true;
		state.mine.write(msg.num, state.mine.now());
	}

	new posts[isThread ? 'Section' : 'Article']({
		// Create model
		model: new posts.models[isThread ? 'Thread' : 'Post'](msg),
		id: msg.num,
		el: el
	});

	if (isThread)
		return;
	var parent = state.posts.get(msg.op);
	if (!parent)
		return;
	parent.get('replies').push(msg.num);
	parent.trigger('shiftReplies');
	// Bump thread to page top
	if (!common.is_sage(msg.email) && bump)
		parent.trigger('bump');
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

	if (model) {
		model.addLinks(links);
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

	if (!model)
		return;
	model.trigger('updateBody', {
		dice: extra && extra.dice,
		links: links || {},
		state: msgState,
		frag: msg[1]
	});
};

// Make the text spoilers toggle revealing on click
main.$doc.on('click', 'del', function (event) {
	if (!event.spoilt) {
		event.spoilt = true;
		$(event.target).toggleClass('reveal');
	}
});

// For mobile
function touchable_spoiler_tag(del) {
	del.html = '<del onclick="void(0)">';
}
exports.touchable_spoiler_tag = touchable_spoiler_tag
main.oneeSama.hook('spoilerTag', touchable_spoiler_tag);

dispatcher[common.FINISH_POST] = function(msg) {
	const num = msg[0];
	delete state.ownPosts[num];
	var model = state.posts.get(num);
	if (model)
		model.set('editing', false);
};

dispatcher[common.DELETE_POSTS] = function(msg) {
	for (let i = 0, lim = msg.length; i < lim; i++) {
		let model = state.posts.get(msg[i]);
		if (model)
			model.remove();

		// TODO: Free up post focus, if any
	}
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

	var model = state.posts.get(op);
	if (model)
		model.remove();
};

dispatcher[common.LOCK_THREAD] = function(msg, op) {
	var model = state.posts.get(op);
	if (model)
		model.set('locked', true);
};

dispatcher[common.UNLOCK_THREAD] = function(msg, op) {
	var model = state.posts.get(op);
	if (model)
		model.set('locked', false);
};

dispatcher[common.DELETE_IMAGES] = function(msg) {
	for (let i = 0, lim = msg.length; i < lim; i++) {
		let model = state.posts.get(msg[i]);
		if (model)
			model.unset('image');
	}
};

dispatcher[common.SPOILER_IMAGES] = function(msg) {
	for (let i = 0, lim = msg.length; i < lim; i++) {
		var model = state.posts.get(msg[i][0]);
		if (model)
			model.trigger('spoiler',msg[i][1]);
	}
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
