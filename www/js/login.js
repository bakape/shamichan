(function () {
	var $ = require('jquery'),
		$script = require('scriptjs');

	var $button = $('<a></a>', {
		href: '#',
		id: 'login-button',
		class: 'persona-button dark',
		css: {'margin-top': '0.5em'}
	});
	var $caption = $('<span>Loading...</span>').appendTo($button);
	$button.appendTo('#identity');

	function inform(msg, color) {
		$caption.text(msg);
		$button.toggleClass('orange', color == 'orange');
		$button.toggleClass('dark', color == 'dark');
	}

	function setup_button() {
		if (!window.loggedInUser) {
			inform('Invoke your Persona', 'orange');
			$button.click(function (event) {
				navigator.id.request();
				event.preventDefault();
			});
		}
		else {
			inform('Logout', 'blue');
			$button.click(function (event) {
				navigator.id.logout();
				event.preventDefault();
			});
		}
		$button.focus();
	}

	function on_login(assertion) {
		inform('Invoking...', 'dark');
		$.ajax({
			type: 'POST',
			url: '../login',
			data: {assertion: assertion},
			dataType: 'json',
			headers: {
				"Access-Control-Allow-Credentials" : true
			},
			success: function (res) {
				if (res && res.status == 'okay') {
					inform('Success!', 'blue');
					window.location.reload();
				}
				else
					inform(res.message||'Unknown error.', 'dark');
			},
			error: function (res) {
				inform('Network error.', 'dark');
				console.error(res);
			}
		});
	}

	function on_logout() {
		inform('Logging out...', 'dark');
		$.ajax({
			type: 'POST',
			url: '../logout',
			data: {csrf: window.x_csrf},
			dataType: 'json',
			headers: {
				"Access-Control-Allow-Credentials" : true
			},
			success: function (res) {
				if (res && res.status == 'okay') {
					inform('Logged out.', 'orange');
					window.location.reload();
				}
				else
					inform(res.message||'Unknown error.', 'dark');
			},
			error: function (res) {
				inform('Network error.', 'dark');
				console.error(res);
			}
		});
	}

	$script('https://login.persona.org/include.js?v=' + clientHash, function () {
		setup_button();
		navigator.id.watch({
			loggedInUser: window.loggedInUser || null,
			onlogin: on_login,
			onlogout: on_logout
		});
	});

	$('<link/>', {
		rel: 'stylesheet',
		href: config.MEDIA_URL + 'css/persona-buttons.css?v=' + clientHash
	})
		.appendTo('head');
})();
