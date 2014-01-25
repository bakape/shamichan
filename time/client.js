(function () {
	$('time').each(function () {
		var t = $(this);
		var d = t.attr('datetime').replace(/-/g, '/'
			).replace('T', ' ').replace('Z', ' GMT');
		t.html(readable_time(new Date(d).getTime()));
	});
})();
