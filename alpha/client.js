/*
 * Handles the brunt of the post-related websocket calls
 */

var $ = require('jquery'),
	common = require('../common'),
	main = require('./main'),
	posts = require('./posts/');

var dispatcher = main.dispatcher;
