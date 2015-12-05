/*
Serve moderation script
 */

let _ = require('underscore'),
	common = require('../../common'),
	express = require('express'),
	resources = require('../state').resources,
	util = require('./util');

const router = module.exports = express.Router();

router.get('/mod.js', function (req, res) {
	// Admin/Moderator privelege is injected on page render and verified
	// serverside. Thus, we can serve the same bundle for both admins and mods.
	if (!common.checkAuth('dj', req.ident))
		return res.sendStatus(404);

	const modJS = resources.modJs;
	if (!modJS)
		return res.sendStatus(500);

	// Not hosted as a file to prevent unauthorised access
	res.set(util.noCacheHeaders);
	res.send(modJS);
});

router.get('/mod.js.map', function (req, res) {
	res.send(resources.modSourcemap);
});
