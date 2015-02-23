(function () {

// Should be part of a greater thread model
var Unread = new Backbone.Model({unreadCount: 0});

// Remember replies that don't need a new desktop notification for 3 days
// Own post are remember for 2 days, so lets keep 1 day as a buffer
var Replies = new Kioku('replies', 3);
var readReplies = Replies.read_all();
Replies.purge_expired_soon();

var normalTitle = document.title;

// Pass visibility changes to Unread model
document.addEventListener('visibilitychange', function (e) {
	var hidden = !!e.target.hidden;
	// Unread post count will reset
	Unread.set({hidden: hidden, unreadCount: 0, reply: !hidden});
	// Prevent scrolling with new posts, if page isn't visible
	autoUnlock(hidden);
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
	// Already read reply || catalog view
	if (readReplies[num]|| CATALOG)
		return;
	if (options.get('notification')) {
		var body = post.get('body');
		var image = post.get('image');
		if((body || image) && document.hidden){
			var n = new Notification('You have been quoted',{
				// if the post doesn't have a image we use a bigger favicon
				icon: encodeURI(mediaURL+ (image ? 'thumb/'+image.thumb : 'css/ui/favbig.png')),
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

// Change the favicon
function favicon(url){
	$('#favicon').remove();
	$('<link/>', {
		type: 'image/x-icon',
		rel: 'shortcut icon',
		href: url,
		id: 'favicon',
	}).appendTo('head');
}

// Needs to be available with no connectivity
var discoFavicon = 'data:image/vnd.microsoft.icon;base64,AAABAAEAEBAAAAEAIABoBAAAFgAAACgAAAAQAAAAIAAAAAEAIAAAAAAAAAQAABMLAAATCwAAAAAAAAAAAAD///8A////AP///wD///8AWUc/AP///wD///8ALikr/y4pKwAuKSv/LikrAFpHOQBWQjUA////AP///wD///8A////AP///wD///8A////AP///wD///8AUFNYAC4pK/8uKSsALikr/y4pKwDT6P0AYlGIAP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wCmm6X/ppul/6abpf+vuuVO////AP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wDT6P3/ppul/6abpf+mm6X/0+j9/////wD///8A////AP///wD///8A////AP///wD///8A////AP///wBQU1j/LCgu//n5+f/5+fn/+fn5/ywoLv9QU1j/////AP///wD///8A////AP///wD///8A////AP///wBQU1j/UFNY//n5+f9qUGD/+fn5/2pQYP/5+fn/UFNY/1BTWP////8A////AP///wD///8A////AP///wD///8AUFNY/1BTWP+nmaX/alBg/2pQYP9qUGD/p5ml/1BTWP9QU1j/////AP///wD///8A////AP///wD///8A////AFBTWP9QU1j/UFNY/x4UIP/T6P3/HhQg/1BTWP9QU1j/UFNY/////wD///8A////AP///wD///8A////AP///wBQU1j/UFNY/9Po/f/T6P3/0+j9/9Po/f/T6P3/UFNY/1BTWP////8A////AP///wD///8A////AP///wD///8AUFNY/9Po/f/T6P3/0+j9/9Po/f/T6P3/0+j9/9Po/f9QU1j/////AP///wD///8A////AP///wD///8A////AFBTWP/T6P3/vJCX/9Po/f/T6P3/0+j9/7yQl//T6P3/UFNY/////wD///8A////AP///wD///8A////AP///wBQU1j/UFNY/3xMUv/T6P3/UFNY/9Po/f98TFL/UFNY/1BTWP////8A////AP///wD///8A////AP///wD///8AUFNY/ycoMv9QU1j/UFNY/1BTWP9QU1j/UFNY/ycoMv9QU1j/////AP///wD///8A////AP///wD///8A////AFBTWP9QU1j/Jygy/ycoMv8nKDL/Jygy/ycoMv9QU1j/UFNY/////wD///8A////AP///wD///8A////AP///wCupYMAUFNY/1BTWP9QU1j/UFNY/1BTWP9QU1j/UFNY/1BTWAD///8A////AP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wD///8A/r8AAP6/AAD+HwAA/B8AAPgPAADwBwAA8AcAAPAHAADwBwAA8AcAAPAHAADwBwAA8AcAAPAHAAD4DwAA//8AAA==';

Unread.on('change', function (model) {
	var attrs = model.attributes;
	var icon = '../favicon.ico';
	if (attrs.alert) {
		document.title = '--- ' + normalTitle;
		return favicon(discoFavicon);
	}
	if (!attrs.hidden) {
		document.title = normalTitle;
		return favicon(icon);
	}
	var prefix = '';
	if (attrs.unreadCount){
		prefix += '(' + attrs.unreadCount + ') ';
		icon = mediaURL + 'css/ui/unreadFavicon.ico';
	}
	if (attrs.reply){
		prefix = '>> ' + prefix;
		icon = mediaURL + 'css/ui/replyFavicon.ico';
	}
	favicon(icon);
	document.title = prefix + normalTitle;
});

})();
