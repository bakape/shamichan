(function () {
	var standalone = !!window.personaExitURL;
	var apiPath = standalone ? '' : '../';

	var $button = $('<a></a>', {
		href: '#',
		id: 'login-button',
		class: 'persona-button dark',
		css: {'margin-top': '0.5em'},
	});
	var $caption = $('<span>Loading...</span>').appendTo($button);
	$button.appendTo(standalone ? 'body' : 'fieldset');

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
			url: apiPath+'login',
			data: {assertion: assertion},
			dataType: 'json',
			success: function (res) {
				if (res && res.status == 'okay') {
					inform('Success!', 'blue');
					setTimeout(return_to_site, 500);
				}
				else
					inform(res.message||'Unknown error.', 'dark');
			},
			error: function (res) {
				inform('Network error.', 'dark');
				console.error(res);
			},
		});
	}

	function on_logout() {
		inform('Logging out...', 'dark');
		$.ajax({
			type: 'POST',
			url: apiPath+'logout',
			data: {csrf: window.x_csrf},
			dataType: 'json',
			success: function (res) {
				if (res && res.status == 'okay') {
					inform('Logged out.', 'orange');
					setTimeout(return_to_site, 1000);
				}
				else
					inform(res.message||'Unknown error.', 'dark');
			},
			error: function (res) {
				inform('Network error.', 'dark');
				console.error(res);
			},
		});
	}

	function return_to_site() {
		if (standalone)
			window.location.href = window.personaExitURL;
		else
			window.location.reload();
	}

	yepnope({
		load: 'https://login.persona.org/include.js',
		complete: function () {
			setup_button();
			navigator.id.watch({
				loggedInUser: window.loggedInUser || null,
				onlogin: on_login,
				onlogout: on_logout,
			});
		},
	});

	$('<link/>', {
		rel: 'stylesheet',
		href: mediaURL + 'css/' + hotConfig.css['persona-buttons.css'],
	})
		.appendTo('head');
})();
