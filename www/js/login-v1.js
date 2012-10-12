(function () {

var $button = $('<a></a>', {
	href: '#',
	id: 'login-button',
	'class': 'persona-button',
	css: {'margin-top': '0.5em'},
});
var $caption = $('<span></span>').appendTo($button);
$button.appendTo('fieldset').focus();

function inform(msg, color) {
	$caption.text(msg);
	$button.toggleClass('orange', color == 'orange');
	$button.toggleClass('dark', color == 'dark');
}

if (!window.loggedInUser) {
	inform('Invoke your Persona', 'orange');
	$button.click(function (event) {
		if (navigator.id)
			navigator.id.request();
		else
			inform('Persona system not loaded.', 'dark');
		event.preventDefault();
	});
}
else {
	inform('Logout', 'blue');
	$button.click(function (event) {
		if (navigator.id)
			navigator.id.logout();
		else
			inform('Persona system not loaded.', 'dark');
		event.preventDefault();
	});
}

function on_login(assertion) {
	inform('Invoking...', 'dark');
	$.ajax({
		type: 'POST',
		url: '../login',
		data: {assertion: assertion},
		dataType: 'json',
		success: function (res) {
			if (res && res.status == 'okay') {
				inform('Success!', 'blue');
				setTimeout(function () {
					window.location.reload();
				}, 500);
			}
			else
				inform(res.message||'Unknown error.', 'dark');
		},
		error: function (res) {
			inform('Network error.', 'dark');
		},
	});
}

function on_logout() {
	inform('Logging out...', 'dark');
	/* defer to client/admin.js */
	logout_admin(function (err) {
		if (err)
			return inform(err, 'dark');
		else {
			inform('Logged out.', 'orange');
			setTimeout(function () {
				window.location.reload();
			}, 1000);
		}
	});
}

yepnope({
	load: 'https://login.persona.org/include.js',
	complete: function () {
		navigator.id.watch({
			loggedInUser: window.loggedInUser || null,
			onlogin: on_login,
			onlogout: on_logout,
		});
	},
});

$('<link></link>', {
	rel: 'stylesheet',
	href: mediaURL + 'css/persona-buttons.css',
}).appendTo('head');

})();
