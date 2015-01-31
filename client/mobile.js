(function(){
	if (!isMobile)
		return;
	// Only comatible with Android Chrome for now
	var n = navigator.userAgent;
	var isChrome = /Chrome/.test(n);
	var isFF = /Firefox/.test(n);
	if (!isChrome)
		return;
	// User refused the first time
	if (localStorage.homescreenRefused)
		return;
	var msg ='To install meguca as a fullscreen webapp, tap the menu button and select ';
	if (isChrome)
		msg += '"Add to home screen"';
	else if (isFF)
		msg += '"Page > Add to Home Screen"';
	msg += '. Tap this message to close. It will not appear again.';
	$('<a/>', {'class': 'mobile'})
		.text('Install Webapp (BETA)')
		.click(function(e){
			new NotificationView(msg);
			$(e.target).remove();
			localStorage.homescreenRefused = true;
		})
		.appendTo($('#banner_center'));
})();
