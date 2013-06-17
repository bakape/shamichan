(function () {

// Should be part of a greater thread model
var Unread = new Backbone.Model({unreadCount: 0});

var normalTitle = document.title;

window.addEventListener('focus', function () {
	Unread.set({blurred: false, unreadCount: 0});
}, false);

window.addEventListener('blur', function () {
	Unread.set({blurred: true, unreadCount: 0});
}, false);

connSM.on('synced', function () {
	Unread.set('alert', false);
});

function dropped() {
	Unread.set('alert', true);
}
connSM.on('dropped', dropped);
connSM.on('desynced', dropped);

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
	var prefix = '';
	if (attrs.alert)
		prefix = '--- ';
	else if (attrs.unreadCount)
		prefix = '(' + attrs.unreadCount + ') ';
	document.title = prefix + normalTitle;
});

})();
