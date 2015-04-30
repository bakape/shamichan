/*
 Resolves the client-side and server-side dependancies, so each one gets only
 what is needed
 */

'use strict';

// Runing on the server
exports.isNode = typeof navigator === 'undefined';

var main;
if (exports.isNode) {
	exports.config = require('../config');
	exports.hotConfig = require('../server/state').hot;
	exports.lang = require('../lang/');
}
else {
	exports.main = main = require('../client/main');
	exports.config = main.config;
	exports.hotConfig = require('../client/state').hotConfig.attributes;
	exports.lang = main.lang;
}
