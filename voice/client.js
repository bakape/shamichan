(function () {

var eleven = /[\u4e00-\u9fbf\u3040-\u309f\u30a0-\u30ff]/;

oneeSama.hook('menuOptions', function (info) {
	// TODO use model instead
	var text = $('#' + info.num).find('blockquote').text();
	if (text && eleven.exec(text))
		info.options.push('Speak');
});

menuHandlers.Speak = function ($post) {
	var $audio = $('<audio/>', {
		src: $post.attr('id') + '/voice',
		attr: {autoplay: 'autoplay'},
	});
	$audio.prop({
		onended: $.proxy($audio, 'remove'),
		onerror: function (err) {
			// TODO
			console.error('audio error', err);
		},
	}).appendTo('body');
};

})();
