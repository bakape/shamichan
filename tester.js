/*
 For testing db.js
 */

const config = require('./config');

// Detect major version and add extra transformers as needed
const tranformers = ['es6.destructuring', 'es6.parameters', 'es6.spread',
	'strict'];
const version = process.version[1];
if (version < 4) {
	tranformers.push('es6.arrowFunctions');
	if (version < 3)
		tranformers.push('es6.properties.computed');
}

// ES6 transpiler require hook. We only enable some not yet implemented
// feature transformers and rely on natives for others.
require('babel/register')({
	// Babel has trouble with hot.js, so we ignore the config module
	ignore: /node_modules|config/,
	sourceMaps: config.DEBUG && 'inline',

	// Stack traces should at least have the exact line numbers displayed
	// correctly
	retainLines: true,
	whitelist: tranformers
});

const db = require('./db');

db.init(err => {
	if (err)
		throw err
});
