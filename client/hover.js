/*
 * Hover previews
 */

var main = require('./main'),
	Backbone = require('backbone');

// Centralised mouseover target tracking
var mouseover = exports.mouseover = new Backbone.Model({target: null});

if (!main.isMobile) {
	main.$doc.on('mouseover', function(e) {
		mouseover.set('target', e.target);
	});
}