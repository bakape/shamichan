/*
 For testing db.js
 */
'use strict'

const config = require('./config')

// Detect major version and add extra transformers as needed
const tranformers = [
	'transform-es2015-destructuring', 'transform-es2015-parameters',
	'transform-strict-mode', 'transform-async-to-generator'
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
})

const db = require('./db')
db.init().catch(err => {throw err})
