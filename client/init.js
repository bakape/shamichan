var $DOC = $(document);
var $name = $('input[name=name]'), $email = $('input[name=email]');
var $ceiling = $('hr:first');

DEFINES.PAGE_BOTTOM = -1;
var menuOptions = ['Focus'];
var menuHandlers = {};

var oneeSama = new OneeSama(function (num) {
	var frag;
	if (this.links && num in this.links) {
		var op = this.links[num];
		var post = Threads.lookup(num, op);
		var desc = post && post.get('mine') && '(You)';
		frag = this.post_ref(num, op, desc);
	}
	else
		frag = '>>' + num;
	this.callback(frag);
});
oneeSama.full = oneeSama.op = THREAD;
