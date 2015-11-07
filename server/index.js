/*
Server entry point
 */

// Only explicity set here . All other modules down the require chain use
// the babel.js strict transformer.
'use strict';

if (!process.getuid())
	throw new Error("Refusing to run as root.");

const config = require('../config'),
	opts = require('./opts'),
	winston = require('winston');

// Read command line arguments. Modifies ../config, so loaded right after it.
opts.parse_args();

// Some config defaults for backwards compatibility.
// TODO: Centralised config defaulting in a later version
if (!config.link_boards)
	config.link_boards = [];

// Build an object of all possible board-like link targets
const targets = config.link_targets = {};
for (let board of config.BOARDS) {
	targets[board] = `../${board}/`;
}
for (let board of config.PSUEDO_BOARDS.concat(config.link_boards)) {
	targets[board[0]] = board[1];
}

// More verbose logging
if (config.DEBUG) {
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


// Detect major version and add extra transformers as needed
const tranformers = [
	'transform-es2015-destructuring', 'transform-es2015-parameters',
	'transform-strict-mode'
]
const version = +process.version.match(/^v(\d+)\./)[1]
const features = {
	5: 'transform-es2015-spread',
	4: 'transform-es2015-arrow-functions',
	3: 'transform-es2015-computed-properties'
}
for (let i = version; i >= 3; i--) {
	if (version === i)
		break;
	tranformers.push(features[i])
}

// ES6 transpiler require hook. We only enable some not yet implemented
// feature transformers and rely on natives for others.
require('babel-core/register')({
	// Babel has trouble with hot.js, so we ignore the config module
	ignore: /node_modules|config/,
	sourceMaps: config.DEBUG && 'inline',
	
	// Stack traces should at least have the exact line numbers displayed
	// correctly
	retainLines: true,
	plugins: tranformers
});

// Require the actual server
require('./server');
