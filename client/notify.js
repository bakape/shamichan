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
let Unread = new Backbone.Model({unreadCount: 0});

// Remember replies that don't need a new desktop notification for 3 days
// Own post are remember for 2 days, so lets keep 1 day as a buffer
let Replies = new memory('replies', 3),
	readReplies = Replies.read_all();
Replies.purge_expired_soon();

const normalTitle = document.title;

// Pass visibility changes to Unread model
document.addEventListener('visibilitychange', function (e) {
	let hidden = e.target.hidden;
	// Unread post count will reset
	Unread.set({
		hidden: hidden,
		unreadCount: 0,
		reply: !hidden
	});
	// TODO: Prevent scrolling with new posts, if page isn't visible
	//autoUnlock(hidden);
}, false);

main.connSM.on('synced', function () {
	Unread.set('alert', false);
});

function dropped() {
	Unread.set('alert', true);
}
main.connSM.on('dropped', dropped);
main.connSM.on('desynced', dropped);

main.comply('repliedToMe', function (post) {
	const num = post.get('num');
	// Already read reply
	if (readReplies[num])
		return;
	if (options.get('notification')) {
		const body = post.get('body'),
			image = post.get('image');
		if ((body || image) && document.hidden && !main.isMobile) {
			new Notification('You have been quoted', {
				// if the post doesn't have a image we use a bigger favicon
				icon: encodeURI(mediaURL
					+ (image ? 'thumb/' + image.thumb : 'css/ui/favbig.png')
				),
				body: body
			})
				.onclick = function() {
					window.focus();
					location.hash = '#' + num;
				};
		}
	}

	Unread.set({reply: true});
	// Record as already read
	Replies.write(num, Replies.now());
});

main.comply('time:syncwatch', function(time){
	if (!options.get('notification') || !document.hidden)
		return;
	new Notification('Syncwatch Starting',{
		body: 'syncwatch starting in : ' + time + ' seconds'
	})
		.onclick = ()=> window.focus();
});

main.comply('post:inserted', function (model) {
	// It's ours, don't notify unread
	if (model.get('mine'))
		return;
	if (document.hidden)
		Unread.set('unreadCount', Unread.get('unreadCount') + 1);
});

// Change the favicon
function favicon(url){
	$('#favicon').remove();
	$('head').append(main.common.parseHTML
		`<link type="image/x-icon"
			rel="shortcut icon"
			href="${url}"
			id="favicon"
		>
		</link>`
	);
}

// Needs to be available with no connectivity
const discoFavicon = 'data:image/vnd.microsoft.icon;base64,AAABAAEAEBAAAAEAIABoBAAAFgAAACgAAAAQAAAAIAAAAAEAIAAAAAAAAAQAABMLAAATCwAAAAAAAAAAAAD///8A////AP///wD///8AWUc/AP///wD///8ALikr/y4pKwAuKSv/LikrAFpHOQBWQjUA////AP///wD///8A////AP///wD///8A////AP///wD///8AUFNYAC4pK/8uKSsALikr/y4pKwDT6P0AYlGIAP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wCmm6X/ppul/6abpf+vuuVO////AP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wDT6P3/ppul/6abpf+mm6X/0+j9/////wD///8A////AP///wD///8A////AP///wD///8A////AP///wBQU1j/LCgu//n5+f/5+fn/+fn5/ywoLv9QU1j/////AP///wD///8A////AP///wD///8A////AP///wBQU1j/UFNY//n5+f9qUGD/+fn5/2pQYP/5+fn/UFNY/1BTWP////8A////AP///wD///8A////AP///wD///8AUFNY/1BTWP+nmaX/alBg/2pQYP9qUGD/p5ml/1BTWP9QU1j/////AP///wD///8A////AP///wD///8A////AFBTWP9QU1j/UFNY/x4UIP/T6P3/HhQg/1BTWP9QU1j/UFNY/////wD///8A////AP///wD///8A////AP///wBQU1j/UFNY/9Po/f/T6P3/0+j9/9Po/f/T6P3/UFNY/1BTWP////8A////AP///wD///8A////AP///wD///8AUFNY/9Po/f/T6P3/0+j9/9Po/f/T6P3/0+j9/9Po/f9QU1j/////AP///wD///8A////AP///wD///8A////AFBTWP/T6P3/vJCX/9Po/f/T6P3/0+j9/7yQl//T6P3/UFNY/////wD///8A////AP///wD///8A////AP///wBQU1j/UFNY/3xMUv/T6P3/UFNY/9Po/f98TFL/UFNY/1BTWP////8A////AP///wD///8A////AP///wD///8AUFNY/ycoMv9QU1j/UFNY/1BTWP9QU1j/UFNY/ycoMv9QU1j/////AP///wD///8A////AP///wD///8A////AFBTWP9QU1j/Jygy/ycoMv8nKDL/Jygy/ycoMv9QU1j/UFNY/////wD///8A////AP///wD///8A////AP///wCupYMAUFNY/1BTWP9QU1j/UFNY/1BTWP9QU1j/UFNY/1BTWAD///8A////AP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wD///8A////AP///wD///8A/r8AAP6/AAD+HwAA/B8AAPgPAADwBwAA8AcAAPAHAADwBwAA8AcAAPAHAADwBwAA8AcAAPAHAAD4DwAA//8AAA==';

Unread.on('change', function (model) {
	const attrs = model.attributes;
	let icon = '../favicon.ico';
	if (attrs.alert) {
		document.title = '--- ' + normalTitle;
		return favicon(discoFavicon);
	}
	else if (!attrs.hidden) {
		document.title = normalTitle;
		return favicon(icon);
	}
	let prefix = '';
	if (attrs.unreadCount) {
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
