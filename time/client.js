(function () {

var readable_time = oneeSama.readable_time;
var relative_time = oneeSama.relative_time;
var rTime = oneeSama.rTime;

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

var is_skewed = (function(){
	var el = document.querySelector('time');
	if (!el)
		return false;
	var d = date_from_time_el(el);
	return readable_time(d.getTime()) != el.innerHTML;
})();

if (is_skewed) {
	if (!rTime)
		adjust_all_times();

	setTimeout(function () {
		// next request, have the server render the right times
		var tz = -new Date().getTimezoneOffset() / 60;
		$.cookie('timezone', tz, { expires: 90 });
	}, 3000);
}

// Replace with relative post timestamps
if (rTime){
	$('time').each(function(){
		var time = date_from_time_el(this).getTime();
		$(this)
			.attr('title', readable_time(time))
			.text(relative_time(time, new Date().getTime()));
	});
	// Regenerate timestamp each minute
	(function increment_time(){
		setTimeout(function(){
			$('time').each(function(){
				var time = date_from_time_el(this).getTime();
				$(this).text(relative_time(time, new Date().getTime()));
			});
			increment_time();
		} ,60000);
	})();
}

})();
