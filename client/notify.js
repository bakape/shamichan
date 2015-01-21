(function () {

// Should be part of a greater thread model
var Unread = new Backbone.Model({unreadCount: 0});

// Remember replies that don't need a new desktop notification for 30 days
var Replies = new Kioku('replies', 30);
var readReplies = Replies.read_all();
Replies.purge_expired_soon();

var normalTitle = document.title;

window.addEventListener('focus', function () {
	Unread.set({unreadCount: 0, reply: false});
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
	var num = post.get('num');
	// Already read reply
	if (readReplies[num])
		return;
	if (options.get('notification')) {
		var body = post.get('body');
		var image = post.get('image');
		if((body || image) && document.hidden){
			var n = new Notification('You have been quoted',{
				// if the post doesn't have a image we use a bigger favicon
				icon: encodeURI(mediaURL+ (image ? 'thumb/'+image.thumb : '/css/ui/favbig.png')),
				body: body,
			});
			n.onclick = function(){
				window.focus();
				location.hash = '#'+num;
			};
		}
	}

	Unread.set({reply: true});
	// Record as already read
	Replies.write(num, Replies.now());
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
	if (document.hidden)
		Unread.set('unreadCount', Unread.get('unreadCount') + 1);
});

Unread.on('change', function (model) {
	var attrs = model.attributes;
	if (document.hidden) {
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
