(function () {

var eleven = /[\u4e00-\u9fbf\u3040-\u309f\u30a0-\u30ff]/;

oneeSama.hook('menuOptions', function (info) {
	// TODO use model instead
	var text = $('#' + info.num).find('blockquote').text();
	if (text && eleven.exec(text))
		info.options.push('Speak');
});

menuHandlers.Speak = function (num) {
	var $audio = $('<audio/>', {
		src: num + '/voice',
		attr: {autoplay: 'autoplay'},
	});
	var a = $audio[0];
	if (a.canPlayType && !a.canPlayType('audio/mpeg;').replace(/no/, '')) {
		alert("Can't play in this browser, sorry.");
		return;
	}
	a.addEventListener('ended', function () {
		$audio.remove();
	}, false);
	// TODO notify error
	$audio.appendTo('body');
};

})();
