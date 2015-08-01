/*
 * This file has no real purpose aside from easing requiring server-side.
 * No extra code modification is required thanks to this. Just paste in your
 * language pack and add it to the LANGS array in ./config.js
 */

var config = require('../config');

for (let i = 0, l = config.LANGS.length; i < l; i++) {
	let lang = config.LANGS[i];
	exports[lang] = require('./' + lang + '/server');
}
