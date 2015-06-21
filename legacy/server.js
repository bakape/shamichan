/*
Yet unported dispatcher functions
 */

dispatcher[common.DELETE_POSTS] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.kotowaru(Muggle("Couldn't delete."));
	/* Omit to-be-deleted posts that are inside to-be-deleted threads */
	let ops = {},
		OPs = db.OPs;
	for (let i = 0, l = nums.length; i < l; i++) {
		let num = nums[i];
		if (num == OPs[num])
			ops[num] = 1;
	}
	nums = nums.filter(function (num) {
		var op = OPs[num];
		return op == num || !(OPs[num] in ops);
	});

	client.db.remove_posts(nums, function (err, dels) {
		if (err)
			client.kotowaru(Muggle("Couldn't delete.", err));
	});
});

dispatcher[common.LOCK_THREAD] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.kotowaru(Muggle("Couldn't (un)lock thread."));
	nums = nums.filter(function (op) { return db.OPs[op] == op; });
	async.forEach(nums, client.db.toggle_thread_lock.bind(client.db),
		function (err) {
			if (err)
				client.kotowaru(Muggle(
					"Couldn't (un)lock thread.", err));
		});
});

dispatcher[common.DELETE_IMAGES] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.kotowaru(Muggle("Couldn't delete images."));
	client.db.remove_images(nums, function (err, dels) {
		if (err)
			client.kotowaru(Muggle("Couldn't delete images.",err));
	});
});

dispatcher[common.SPOILER_IMAGES] = caps.mod_handler(function (nums, client) {
	if (!inactive_board_check(client))
		return client.kotowaru(Muggle("Couldn't spoiler images."));
	client.db.force_image_spoilers(nums, function (err) {
		if (err)
			client.kotowaru(Muggle("Couldn't spoiler images.",
				err));
	});
});

dispatcher[common.EXECUTE_JS] = function (msg, client) {
	if (!caps.can_administrate(client.ident))
		return false;
	if (!check(['id'], msg))
		return false;
	var op = msg[0];
	client.db.set_fun_thread(op, function (err) {
		if (err)
			client.kotowaru(err);
	});
	return true;
};

// Non-persistent global live admin notifications
dispatcher[common.NOTIFICATION] = function(msg, client){
	if (!caps.can_administrate(client.ident))
		return false;
	if (!check(['string'], msg))
		return false;
	okyaku.push([0, common.NOTIFICATION, common.escape_html(msg[0])]);
	return true;
};
