/*
 * Handles the brunt of the post-related websocket calls
 */

const main = require('./main'),
	{_, common, dispatcher, etc, posts, state} = main;

const online = document.query('#onlineCount');

_.extend(dispatcher, {
	[common.INSERT_POST](message) {
		let [msg, bump] = message;
		bump = bump && state.page.get('live');
		const isThread = !msg.op,
			{num} = msg;
		if (isThread)
			state.syncs[num] = 1;
		msg.editing = true;

		// Did I create this post?
		let el;
		const {nonce} = msg;
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
			const postForm = main.request('postForm');
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
		const model = new posts.models[isThread ? 'Thread' : 'Post'](msg);
		const view = new posts[isThread ? 'Section' : 'Article']({model, el,
			id: num});
		if (!el)
			view.render().insertIntoDOM();
		view.clientInit();

		checkRepliedToMe(msg.links, num);
		main.request('post:inserted', model);

		if (isThread)
			return;
		const parent = state.posts.get(msg.op);
		if (!parent)
			return;
		parent.get('replies').push(num);
		if (state.page.get('thread'))
			return;
		parent.dispatch('shiftReplies');

		// Bump thread to page top
		if (bump)
			parent.dispatch('bumpThread');
	},
	[common.INSERT_IMAGE](msg) {
		const [num, img] = msg;

		// Did I just upload this?
		const postModel = main.request('postModel'),
			toPostForm = postModel && postModel.get('num') == num;
		if (toPostForm)
			main.request('postForm').insertUploaded(img);

		// If the image gets inseted into the postForm, we don't need the
		// generic model to fire a separate image render
		modelHandler(num, model => model.setImage(img, toPostForm));
	},
	[common.UPDATE_POST](msg) {
		const [num, frag, extra] = msg;
		modelHandler(num, model => {
			const {links} = extra;
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
		const [num] = msg;
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
	[common.LOCK_THREAD](msg) {
		modelHandler(msg[0], model => model.toggleLocked(true, msg[1]));
	},
	[common.UNLOCK_THREAD](msg) {
		modelHandler(msg[0], model => model.toggleLocked(false, msg[1]));
	},
	[common.DELETE_IMAGES](msg) {
		modelHandler(msg[0], model => model.removeImage(msg[1]));
	},
	[common.SPOILER_IMAGES](msg) {
		modelHandler(msg[0], model => model.setSpoiler(msg[1], msg[2]));
	},
	[common.BAN](msg) {
		// Only a 0 is passed to unauthenticated clients, if the ban was not
		// set to be displayed publicly. Otherwise a post number. A side
		// effect of complying to the existing pub/sub spec. Authenticated
		// staff receive either a post number or 0 and detailed ban information.
		let [num, info] = msg;
		if (!num) {
			if (!info)
				return;
			info = JSON.parse(info);
			num = info.num;
		}
		modelHandler(num, model => model.setBan(num, info));
	},
	[common.BACKLINK](msg) {
		modelHandler(msg[0], model => model.addBacklink(msg[1], msg[2]));
	},
	[common.ONLINE_COUNT](msg) {
		online.textContent = '['+msg[0]+']';
	},
	// Sync settings with server
	[common.HOT_INJECTION](msg) {
		const [force, hash, hotConfig] = msg;

		// Request new varibles, if hashes don't match
		if (!force && hash !== state.configHash)
			main.send([common.HOT_INJECTION, true]);
		// Update variables and hash
		else if (force) {
			state.configHash = hash;
			state.hotConfig.set(hotConfig);
		}
	}
});

dispatcher[common.SYNCHRONIZE] = main.connSM.feeder('sync');
dispatcher[common.INVALID] = main.connSM.feeder('invalid');

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
	const model = state.posts.get(num);
	if (model)
		func(model);
}

// Make the text spoilers toggle revealing on click
etc.listener(document, 'click', 'del', function (event) {
	if (event.spoilt)
		return;
	event.spoilt = true;
	event.target.classList.toggle('reveal');
});
