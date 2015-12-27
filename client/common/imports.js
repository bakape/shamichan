/*
 Resolves the client-side and server-side dependancies, so each one gets only
 what is needed
 */

// Runing on the server
exports.isNode = typeof navigator === 'undefined';

if (exports.isNode) {
	exports.config = require('../config');
	exports.hotConfig = require('../server/state').hot;
}
else {
	const main = exports.main = require('main');
	exports.config = main.config;
	exports.hotConfig = main.state.hotConfig.attributes;
}
