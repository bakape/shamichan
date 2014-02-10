(function () {

var eleven = /[\u4e00-\u9fbf\u3040-\u309f\u30a0-\u30ff]/;

oneeSama.hook('menuOptions', function (info) {
	// TODO use model instead
	if (!info.model)
		return;
	var text = $('#' + info.model.id).find('blockquote').text();
	if (text && eleven.exec(text))
		info.options.push('Speak');
});

menuHandlers.Speak = function (model) {
	var $audio = $('<audio/>', {
		src: model.id + '/voice',
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
