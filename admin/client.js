/*
Client-side administration logic
 */

let main = require('main');

const ident = window.IDENT;

// Pass login status to ./www/js/login.js
window.loggedInUser = ident.email;
window.x_csrf = ident.csrf;

alert('Install Gentoo');
