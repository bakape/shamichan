// Remember replies that don't need a new desktop notification for 3 days
// Own post are remember for 2 days, so lets keep 1 day as a buffer
let replies = new main.Memory('replies', 3);

main.reply('repliedToMe', function (num) {
	let post = state.posts.get(num);
	if (!post)
		return;
	post = post.attributes;

	// Already displayed a notification for the reply. Needs to be read
	// freshly from local storage each time, not to trigger multiple times,
	// if the same post is displayed in multiple tabs.
	if (num in replies.readAll())
		return;
	if (options.get('notification') && document.hidden && !main.isMobile) {
		let n = new Notification(main.lang.quoted, {
			// if the post doesn't have a image we use a bigger favicon
			icon: (post.image && options.get('thumbs')!=='hide' && !main.oneeSama.workMode) ? main.oneeSama.thumbPath(post.image)
				: '/ass/css/ui/favbig.png',
			body: post.body
		});
		n.onclick = function() {
			window.focus();
			location.hash = '#p' + num;
		};
	}

	notify.set({reply: true});
	// Record as already notified/read to local storage
	replies.write(num);
});

main.reply('time:syncwatch', function () {
	if (!options.get('notification') || !document.hidden)
		return;
	new Notification(main.lang.syncwatchStarting)
		.onclick = () => window.focus();
});
