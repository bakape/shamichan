/*
 * Background controller. Wallpapers, proper fitting and video backgrounds
 */

var Backbone = require('backbone');

// TODO: write it
module.exports = new Backbone.View({
	el: document.getElementById('user_bg'),
	model: new Backbone.Model({
		id: 'background'
	}),
	set: function(toggle) {

	},
	glass: function() {

	},
	genCustom: function(file) {

	}
});
