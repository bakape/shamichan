/*
 Unread post etc. notifications
 */

let $ = require('jquery'),
	Backbone = require('backbone'),
	main = require('./main'),
	memory = require('./memory'),
	state = main.state,
	options = main.options;

const mediaURL = main.config.MEDIA_URL;

// Needs to be available with no connectivity, so we download and cache it
let discoFavicon = '';
{
	// jQuery does not support XHR2 binary data type request
	let xhr = new XMLHttpRequest();
	xhr.open('GET', config.SECONDARY_MEDIA_URL + 'css/ui/disconnected.ico');
	xhr.responseType = 'blob';
	xhr.onload = function() {
		if (this.status === 200)
			discoFavicon = window.URL.createObjectURL(this.response);
	};
	xhr.send();
}

let NotifyModel = Backbone.Model.extend({
	initialize: function () {
		this.$favicon = $('#favicon');
		this.check(this);

		this.listenTo(this, 'change', this.check);
		main.comply('post:inserted', model => {
			// It's ours, don't notify unread
			if (model.get('mine'))
				return;
			if (document.hidden)
				this.set('unreadCount', this.get('unreadCount') + 1);
		});
		main.comply('notify:title', title => this.set('title', title));

		// Pass visibility changes to notify model
		document.addEventListener('visibilitychange', e => {
			const hidden = e.target.hidden;
			// Unread post count will reset
			this.set({
				hidden: hidden,
				unreadCount: 0,
				reply: !hidden
			});
			// Prevent scrolling with new posts, if page isn't visible
			if (!options.get('alwaysLock')) {
				main.command(
					'scroll:focus',
					hidden && main.$threads.find('article').last().attr('id')
				);
			}
		}, false);

		let dropped = () => this.set('alert', true);
		main.connSM.on('dropped', dropped);
		main.connSM.on('desynced', dropped);
		main.connSM.on('synced', () => notify.set('alert', false));
	},

	check: function (model) {
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

	render: function(icon, prefix) {
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
let replies = new memory('replies', 3);

main.comply('repliedToMe', function (post) {
	post = post.attributes;
	const num = post.num;
	// Already displayed a notification for the reply. Needs to be read
	// freshly from local storage each time, not to trigger multiple times,
	// if the same post is displayed in multiple tabs.
	if (replies.read_all()[num])
		return;
	if (options.get('notification') && document.hidden && !main.isMobile) {
		new Notification('You have been quoted', {
			// if the post doesn't have a image we use a bigger favicon
			icon: post.image ? main.oneeSama.thumbPath(data)
				: mediaURL + 'css/ui/favbig.png',
			body: post.body
		})
			.onclick = function() {
				window.focus();
				location.hash = '#' + num;
			};
	}

	notify.set({reply: true});
	// Record as already notified/read to local storage
	replies.write(num, replies.now());
});

main.comply('time:syncwatch', function(time){
	if (!options.get('notification') || !document.hidden)
		return;
	new Notification('Syncwatch Starting', {
		body: 'syncwatch starting in : ' + time + ' seconds'
	})
		.onclick = () => window.focus();
});
