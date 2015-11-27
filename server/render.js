const _ = require('underscore'),
	common = require('../common'),
	config = require('../config'),
	languagePacks = require('../lang'),
	state = require('./state');

/**
 * Inject dynamic content into precompiled templates
 * @param {http.ClientRequest} req
 * @param {Object|null} json
 * @returns {string}
 */
export default function (req, json) {
	const {isMobile, isRetarded, ident} = req,
		template = RES[`${isMobile ? 'mobile' : 'index'}Tmpl-${req.lang}`],
		lang = languagePacks[req.lang]
	let html = template[0]
		+ JSON.stringify(json)
		+ template[1]
	if (isRetarded)
		html += retardBanner(lang)
	html += template[2]
	if (!isMobile)
		html += imageBanner()
	html += template[3]
	if (ident.auth)
	    html += loginCredentials(ident)
	return html + template[4]
}

/**
 * Notify the user, he/she/it should consider a brain transplant, by inserting
 * a banner urging to change their browser
 * @param {string} lang
 * @returns {string}
 */
function retardBanner(lang) {
	let html = `<div class="retardedBrowser">`
		+ languagePacks[lang].worksBestWith
		+ ' '
	for (let browser of ['chrome', 'firefox', 'opera']) {
		html += `<img src="${config.MEDIA_URL}css/ui/${browser}.png">`
	}
	return html + '</div>'
}

/**
 * Render image banner
 * @returns {string}
 */
function imageBanner() {
	const banners = state.hot.BANNERS
	if (!banners)
		return ''
	return `<img src="${config.MEDIA_URL}banners/${common.random(banners)}">`
}

/**
 * Inject staff login credentials. These will be used to download the moderation
 * JS client bundle.
 * @param {Object} ident
 * @returns {string}
 */
function loginCredentials(ident) {
	const keys = JSON.stringify(_.pick(ident, 'auth', 'csrf', 'email'))
	return `var IDENT = ${keys}`
}
