(function () {

// Should be part of a greater thread model
var Unread = new Backbone.Model({unreadCount: 0});

var normalTitle = document.title;

window.addEventListener('focus', function () {
	Unread.set({blurred: false, unreadCount: 0, reply: false});
}, false);

window.addEventListener('blur', function () {
	Unread.set({blurred: true, unreadCount: 0, reply: false});
}, false);

connSM.on('synced', function () {
	Unread.set('alert', false);
});

function dropped() {
	Unread.set('alert', true);
}
connSM.on('dropped', dropped);
connSM.on('desynced', dropped);

Backbone.on('repliedToMe', function (post) {
	if (options.get('notification')) {
		var body = post.get('body');
		var image = post.get('image');
		if((body || image) && document.hidden)
			new Notification('You have been quoted',{
				// if the post doesn't have a image we us a bigger favicon
				icon: encodeURI(mediaURL+ (image ? 'thumb/'+image.thumb : '/css/ui/favbig.png')),
				body: body,
			});
	}

	Unread.set({reply: true});
});
Backbone.on('syncCountdown', function(time){
	if (options.get('notification')) {
		if(document.hidden)
			new Notification('Syncwatch Starting',{
				body: 'syncwatch starting in : '+time+' seconds',
			});
	}
});
Backbone.on('afterInsert', function (model) {
	if (model && model.get('mine'))
		return; // It's ours, don't notify unread
	if (Unread.get('blurred'))
		Unread.set('unreadCount', Unread.get('unreadCount') + 1);
});

Unread.on('change', function (model) {
	var attrs = model.attributes;
	if (!attrs.blurred) {
		document.title = normalTitle;
		return;
	}
	if (attrs.alert) {
		document.title = '--- ' + normalTitle;
		return;
	}

	var prefix = '';
	if (attrs.reply)
		prefix += '>> ';
	if (attrs.unreadCount)
		prefix += '(' + attrs.unreadCount + ') ';

	document.title = prefix + normalTitle;
});

})();
