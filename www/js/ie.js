var h5s = ['abbr', 'aside', 'article', 'code', 'figcaption', 'figure',
		'section', 'time'];
for (var i = 0; i < h5s.length; i++)
	document.createElement(h5s[i]);
window.onload = function () { $('h1').after('<h2>Google Chrome recommended.</h2>'); };
