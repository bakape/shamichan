/*
Copies configuration files from examples on `npm install`, if none exist
 */
'use strict';

let fs = require('fs-extra'),
	path = require('path');

const examplePath = path.join('config', 'examples');

function copy(file) {
	const target = path.join('config', file);
	try {
		// Throws error on no file found
		fs.statSync(target);
	}
	catch (e) {
		fs.copySync(path.join(examplePath, file), target);
	}
}

for (let file of fs.readdirSync(examplePath)) {
	copy(file);
}
