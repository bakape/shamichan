/*
 * Handles the brunt of the post-related websocket calls
 */

let main = require('./main'),
	{$, _, common, dispatcher, posts, state} = main;

dispatcher[common.INSERT_POST] = function(msg) {
	let bump = msg[1] && state.page.get('live');
	msg = msg[0];
	const isThread = !msg.op,
		{num} = msg;
	if (isThread)
		state.syncs[num] = 1;
	msg.editing = true;

	// Did I create this post?
	let el;
	const nonce = msg.nonce;
	delete msg.nonce;
	const myNonce = main.request('nonce:get')[nonce];
	if (myNonce && myNonce.tab === state.page.get('tabID')) {
		// posted in this tab; transform placeholder
		state.ownPosts[num] = true;
		main.oneeSama.trigger('insertOwnPost', msg);
		main.postSM.feed('alloc', msg);
		bump = false;

		main.request('nonce:destroy', nonce);
		// if we've already made a placeholder for this post, use it
		let postForm = main.request('postForm');
		if (postForm && postForm.el)
			el = postForm.el;
	}
	
	// Add to my post set. Separate `if`, so posts form other tabs also
	// register.
	if (myNonce) {
		msg.mine = true;
		state.mine.write(num);
	}
	state.addLinks(msg.links);

	// Create model
	let model = new posts.models[isThread ? 'Thread' : 'Post'](msg);
	let view = new posts[isThread ? 'Section' : 'Article']({
		model,
		id: num,
		el
	});
	if (!el)
		view.render().insertIntoDOM();
	view.clientInit();

	checkRepliedToMe(msg.links, num);
	main.request('post:inserted', model);

	if (isThread)
		return;
	let parent = state.posts.get(msg.op);
	if (!parent)
		return;
	parent.get('replies').push(num);
	if (state.page.get('thread'))
		return;
	parent.dispatch('shiftReplies');
	// Bump thread to page top
	if (bump)
		parent.dispatch('bumpThread');
};

// Check if new posts links to one of my posts
function checkRepliedToMe(links, sourceNum) {
	if (!links)
		return;
	const mine = state.mine.readAll();
	for (let num in links) {
		if (num in mine)
			main.request('repliedToMe', sourceNum);
	}
}

// Find model and pass it to function, if it exists
function modelHandler(num, func) {
	let model = state.posts.get(num);
	if (model)
		func(model);
}

let $online = $('#onlineCount');

_.extend(dispatcher, {
	[common.INSERT_IMAGE](msg) {
		const num = msg[0];

		// Did I just upload this?
		let postModel = main.request('postModel'),
			img = msg[1];
		const toPostForm = postModel && postModel.get('num') == num;
		if (toPostForm)
			main.request('postForm').insertUploaded(img);

		// If the image gets inseted into the postForm, we don't need the
		// generic model to fire a separate image render
		modelHandler(num, model => model.setImage(img, toPostForm));
	},
	[common.UPDATE_POST](msg) {
		const num = msg[0],
			frag = msg[1],
			extra = msg[2];
		modelHandler(num, function (model) {
			const links = extra.links;
			state.addLinks(links);
			model.update(frag, extra);
			checkRepliedToMe(links, num);

			// Am I updating my own post?
			if (num in state.ownPosts)
				main.oneeSama.trigger('insertOwnPost', extra);
			else {
				model.dispatch('updateBody', {
					dice: extra.dice,
					state: extra.state,
					frag
				});
			}
		});
	},
	[common.FINISH_POST](msg) {
		const num = msg[0];
		delete state.ownPosts[num];
		modelHandler(num, function (model) {
			// No change event listener to avoid extra overhead
			model.set('editing', false);
			model.dispatch('renderEditing', false);
		});
	},
	[common.DELETE_POSTS](msg) {
		modelHandler(msg[0], model => model.deletePost(msg[1]));
	},
	[common.LOCK_THREAD](msg, op) {
		modelHandler(op, model => model.toggleLocked(true));
	},
	[common.UNLOCK_THREAD](msg, op) {
		modelHandler(op, model => model.toggleLocked(false));
	},
	[common.DELETE_IMAGES](msg) {
		modelHandler(msg[0], model => model.removeImage(msg[1]));
	},
	[common.SPOILER_IMAGES](msg) {
		modelHandler(msg[0], model => model.setSpoiler(msg[1], msg[2]));
	},
	[common.BACKLINK](msg) {
		modelHandler(msg[0], model => model.addBacklink(msg[1], msg[2]));
	},
	[common.ONLINE_COUNT](msg) {
		$online.text('['+msg[0]+']');
	},
	// Sync settings with server
	[common.HOT_INJECTION](msg) {
		// Request new varibles, if hashes don't match
		if (msg[0] == false && msg[1] != state.configHash)
			main.request('send', [common.HOT_INJECTION, true]);
		// Update variables and hash
		else if (msg[0] == true) {
			state.configHash = msg[1];
			state.hotConfig.set(msg[2]);
		}
	}
});

dispatcher[common.SYNCHRONIZE] = main.connSM.feeder('sync');
dispatcher[common.INVALID] = main.connSM.feeder('invalid');

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
