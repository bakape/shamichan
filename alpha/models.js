/*
 * Core Backbone models
 */
var Backbone = require('backbone');

var PostCollection = Backbone.Collection.extend({
	idAttribute: 'num'
});

// All posts currently displayed
var Posts = exports.Posts = new PostCollection();

