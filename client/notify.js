/*
 Unread post etc. notifications
 */

const main = require('./main'),
	{$, Backbone, config, connSM, state, options} = main;

const mediaURL = config.MEDIA_URL;

// Needs to be available with no connectivity, so we download and cache it
let discoFavicon = '',
	xhr = new XMLHttpRequest();
xhr.open('GET', mediaURL + 'css/ui/disconnected.ico');
xhr.responseType = 'blob';
xhr.onload = function() {
	if (this.status === 200)
		discoFavicon = window.URL.createObjectURL(this.response);
};
xhr.send();

let NotifyModel = Backbone.Model.extend({
	initialize() {
		this.$favicon = $('#favicon');
		this.check(this);

		this.listenTo(this, 'change', this.check);
		main.reply('post:inserted', model => {
			// It's ours, don't notify unread
			if (model.get('mine'))
				return;
			if (document.hidden)
				this.set('unreadCount', this.get('unreadCount') + 1);
		});
		main.reply('notify:title', title => this.set('title', title));

		// Pass visibility changes to notify model
		document.addEventListener('visibilitychange', e => {
			const hidden = e.target.hidden;
			// Unread post count will reset
			this.set({
				hidden: hidden,
				unreadCount: 0,
				reply: !hidden
			});
		}, false);

		let dropped = () => this.set('alert', true);
		connSM.on('dropped', dropped);
		connSM.on('desynced', dropped);
		connSM.on('synced', () => notify.set('alert', false));
	},
	check(model) {
		const {hidden, unreadCount, reply, alert} = model.attributes;
		let icon = mediaURL + 'favicon.ico';
		if (alert)
			return this.render(discoFavicon, '--- ');
		else if (!hidden)
			return this.render(icon, '');
		let prefix = '';
		if (unreadCount) {
			prefix += `(${unreadCount}) `;
			icon = mediaURL + 'css/ui/unreadFavicon.ico';
		}
		if (reply){
			prefix = '>> ' + prefix;
			icon = mediaURL + 'css/ui/replyFavicon.ico';
		}
		this.render(icon, prefix);
	},
	render(icon, prefix) {
		document.title = prefix + this.get('title');
		this.$favicon.attr('href', icon);
	}
});

let notify = new NotifyModel({
	unreadCount: 0,
	title: document.title
});

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
				: mediaURL + 'css/ui/favbig.png',
			body: post.body
		});
		n.onclick = function() {
			window.focus();
			location.hash = '#' + num;
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
