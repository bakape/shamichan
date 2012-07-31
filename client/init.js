var $DOC = $(document);
var $name = $('input[name=name]'), $email = $('input[name=email]');
var $ceiling = $('hr:first');

DEFINES.PAGE_BOTTOM = -1;
var menuOptions = ['Focus'];
var menuHandlers = {};

var oneeSama = new OneeSama(function (num) {
	if (this.links && num in this.links)
		this.callback(this.post_ref(num, this.links[num]));
	else
		this.callback('>>' + num);
});
oneeSama.full = oneeSama.op = THREAD;
