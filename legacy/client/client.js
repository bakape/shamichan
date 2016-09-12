_.extend(dispatcher, {
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