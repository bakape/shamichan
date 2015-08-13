/*
Server entry point
 */

// Only explicity set here . All other modules down the require chain use
// the babel.js strict transformer.
'use strict';

if (!process.getuid())
	throw new Error("Refusing to run as root.");

let config = require('../config'),
	winston = require('winston');

// More verbose logging
if (config.DEBUG) {
	require('longjohn');
	winston.remove(winston.transports.Console);
	winston.add(winston.transports.Console, {level: 'verbose'});
	winston.warn("Running in (insecure) debug mode.");
	winston.warn("Do not use on the public internet.");
}
// For production
else {
	winston.remove(winston.transports.Console);
	winston.add(winston.transports.File, {
		level: 'error',
		filename: 'error.log',
		handleExceptions: true
	});
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
	whitelist: [
		'es6.arrowFunctions',
		'es6.destructuring',
		'es6.parameters',
		'es6.properties.computed',
		'es6.spread',
		'strict'
	]
});

// Read command line arguments. Modifies ../configure, so loaded right after it.
let opts = require('./opts');
if (require.main == module)
	opts.parse_args();
opts.load_defaults();

// Require the actual server
require('./server');
