/*
 * Handles the brunt of the post-related websocket calls
 */

let main = require('./main'),
	{$, common, dispatcher, posts, state} = main;

dispatcher[common.INSERT_POST] = function(msg) {
	let bump = msg[1] && state.page.get('live');
	msg = msg[0];
	const isThread = !msg.op;
	if (isThread)
		state.syncs[msg.num] = 1;
	msg.editing = true;

	// Did I create this post?
	var el;
	const nonce = msg.nonce;
	delete msg.nonce;
	const myNonce = main.request('nonce:get')[nonce];
	if (myNonce && myNonce.tab === state.page.get('tabID')) {
		// posted in this tab; transform placeholder
		state.ownPosts[msg.num] = true;
		main.oneeSama.trigger('insertOwnPost', msg);
		main.postSM.feed('alloc', msg);
		bump = false;

		main.command('nonce:destroy', nonce);
		// if we've already made a placeholder for this post, use it
		let postForm = main.request('postForm');
		if (postForm && postForm.el)
			el = postForm.el;
	}
	// Add to my post set
	if (myNonce) {
		msg.mine = true;
		state.mine.write(msg.num, state.mine.now());
	}

	// Create model
	let model = new posts.models[isThread ? 'Thread' : 'Post'](msg);
	new posts[isThread ? 'Section' : 'Article']({
		model: model,
		id: msg.num,
		el: el
	})
		.clientInit();

	main.command('post:inserted', model);

	if (isThread)
		return;
	let parent = state.posts.get(msg.op);
	if (!parent)
		return;
	parent.get('replies').push(msg.num);
	if (state.page.get('thread'))
		return;
	parent.dispatch('shiftReplies');
	// Bump thread to page top
	if (bump)
		parent.dispatch('bumpThread');
};

dispatcher[common.INSERT_IMAGE] = function(msg) {
	const num = msg[0];
	let model = state.posts.get(num);
	// Did I just upload this?
	let postModel = main.request('postModel'),
		img = msg[1];
	const toPostForm = postModel && postModel.get('num') == num;
	if (toPostForm)
		main.request('postForm').insertUploaded(img);
	// If the image gets inseted into the postForm, we don't need the
	// generic model to fire a separate image render
	if (model)
		model.setImage(img, toPostForm);
};

dispatcher[common.UPDATE_POST] = function(msg) {
	const num = msg[0],
		links = msg[4],
		msgState = [msg[2] || 0, msg[3] || 0];
	var extra = msg[5],
		model = state.posts.get(num);

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

	if (!model)
		return;
	model.dispatch('updateBody', {
		dice: extra && extra.dice,
		links: links || {},
		state: msgState,
		frag: msg[1]
	});
};

dispatcher[common.FINISH_POST] = function(msg) {
	const num = msg[0];
	delete state.ownPosts[num];
	var model = state.posts.get(num);
	if (model) {
		// No change event listener to avoid extra overhead
		model.set('editing', false);
		model.dispatch('renderEditing', false);
	}
};

dispatcher[common.DELETE_POSTS] = function(msg) {
	for (let i = 0, lim = msg.length; i < lim; i++) {
		let model = state.posts.get(msg[i]);
		if (model)
			model.remove();
	}
};

dispatcher[common.DELETE_THREAD] = function(msg, op) {
	delete state.syncs[op];
	delete state.ownPosts[op];

	let postModel = main.request('postModel');
	if (postModel) {
		const num = postModel.get('num');
		if ((postModel.get('op') || num) === op)
			main.postSM.feed('done');
		if (num === op)
			return;
	}

	var model = state.posts.get(op);
	if (model)
		model.remove();
};

dispatcher[common.LOCK_THREAD] = function(msg, op) {
	let model = state.posts.get(op);
	if (model)
		model.toggleLocked(true);
};

dispatcher[common.UNLOCK_THREAD] = function(msg, op) {
	let model = state.posts.get(op);
	if (model)
		model.toggleLocked(false);
};

dispatcher[common.DELETE_IMAGES] = function(msg) {
	for (let i = 0, lim = msg.length; i < lim; i++) {
		let model = state.posts.get(msg[i]);
		if (model)
			model.removeImage();
	}
};

dispatcher[common.SPOILER_IMAGES] = function(msg) {
	for (let i = 0, lim = msg.length; i < lim; i++) {
		const spoiler = msg[i];
		let model = state.posts.get(spoiler[0]);
		if (!model)
			continue;
		model.setSpoiler(spoiler[1]);
	}
};

dispatcher[common.BACKLINK] = function(msg) {
	let model = state.posts.get(msg[0]);
	if (model)
		model.addBacklink(msg[1], msg[2]);
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
		main.command('send', [common.HOT_INJECTION, true]);
	// Update variables and hash
	else if (msg[0] == true) {
		state.configHash = msg[1];
		state.hotConfig.set(msg[2]);
	}
};

// Make the text spoilers toggle revealing on click
main.$doc.on('click', 'del', function (event) {
	if (!event.spoilt) {
		event.spoilt = true;
		$(event.target).toggleClass('reveal');
	}
});

/*
 * TODO: These are used only for the Admin panel. Would be nice, if we could
 * set those in admin/client.js. Would need to export main.js outside the bundle
 * then.
 */
dispatcher[common.MODEL_SET] = function (msg, op) {};
dispatcher[common.COLLECTION_RESET] = function (msg, op) {};
dispatcher[common.COLLECTION_ADD] = function (msg, op) {};
