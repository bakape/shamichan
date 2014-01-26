(function () {

function adjust_all_times() {
	$('time').each(function () {
		var t = $(this);
		var d = t.attr('datetime').replace(/-/g, '/'
			).replace('T', ' ').replace('Z', ' GMT');
		t.html(readable_time(new Date(d).getTime()));
	});
}

adjust_all_times();

})();
