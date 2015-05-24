/*
 * Various mobile-only code
 */

let $ = require('jquery'),
	main = require('./main');

(function(){
	// User refused the first time
	if (!main.isMobile || localStorage.homescreenRefused)
		return;
	// Only comatible with Android Chrome for now
	const n = navigator.userAgent,
		isChrome = /Chrome/.test(n),
		isFF = /Firefox/.test(n);
	// Only Chrome works for now
	if (!isChrome)
		return;
	let msg ='To install meguca as a fullscreen webapp, tap the menu button'
		+ ' and select ';
	if (isChrome)
		msg += '"Add to home screen"';
	else if (isFF)
		msg += '"Page > Add to Home Screen"';
	msg += '. Tap this message to close. It will not appear again.';

	$('<a class="mobile">Install Webapp (BETA)</a>')
		.click(function(e){
			main.command('notification', msg);
			e.target.remove();
			localStorage.homescreenRefused = true;
		})
		.appendTo('#banner_center');
})();
