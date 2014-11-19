(function () {

ComposerView.prototype.word_filter = function (words) {
	return words.replace(/(\w+)/g, function (orig) {
		var word = {
			nope: 'nope',
		}[orig.toLowerCase()];
		if (word && word.indexOf(',') >= 0) {
			word = word.split(',');
			word = word[Math.floor(Math.random() * word.length)];
		}
		return (word || orig).replace('>', '<');
	});
};

var $b = $('body');
var back = $b.css('background');
$b.css('background', 'black');
setTimeout(function () { $b.css('background', back); }, 200);

})();
