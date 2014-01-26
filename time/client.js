(function () {

var readable_time = oneeSama.readable_time;

function adjust_all_times() {
	$('time').each(function () {
		var date = date_from_time_el(this);
		this.innerHTML = readable_time(date.getTime());
	});
}

function date_from_time_el(el) {
	var d = el.getAttribute('datetime').replace(/-/g, '/'
		).replace('T', ' ').replace('Z', ' GMT');
	return new Date(d);
}

function is_skewed() {
	var el = document.querySelector('time');
	if (!el)
		return false;
	var d = date_from_time_el(el);
	return readable_time(d.getTime()) != el.innerHTML;
}

if (is_skewed()) {
	adjust_all_times();

	setTimeout(function () {
		// next request, have the server render the right times
		var tz = -new Date().getTimezoneOffset() / 60;
		$.cookie('timezone', tz, { expires: 90 });
	}, 3000);
}

})();
