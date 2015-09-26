/*
Client-side administration logic
 */

const main = require('main'),
	{$, config} = main;

//XXX: This module in general is not very DRY. Need to refactor later.

// Only used to affect some client rendering practises. Anything actually
// needing security has stricter authorisation checks.
const ident = main.ident = window.IDENT;

// Pass login status to ./www/js/login.js
window.loggedInUser = ident.email;
window.x_csrf = ident.csrf;

// Container for all moderation modals
main.$overlay = $('<div id="modOverlay"></div>').appendTo('body');

// Open modal map
main.modals = {};

require('./title');
require('./toolbox');

$('<link/>', {
	rel: 'stylesheet',
	href: `${config.MEDIA_URL}css/mod.css?v=${main.cssHash}`
}).appendTo('head');

// Add staff board to board navigation
const staff = config.STAFF_BOARD;
$('#navTop')
	.children('a')
	.last()
	.after(` / <a href="../${staff}/" class="history">${staff}</a>`);
