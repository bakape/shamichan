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
	var dTime = el.getAttribute('datetime');
	// Don't crash the function, if scanning an unsynced post in progress
	if (!dTime)
		return new Date();
	var d = dTime.replace(/-/g, '/'
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
				$(this).innerHTML(relative_time(time, new Date().getTime()));
			});
			increment_time();
		} ,60000);
	})();
}

})();

function timer_from_el(el) {
	var now = Date.now()
	var start= el.getAttribute('start');
	var diff=now-start;
	var hour = Math.floor(diff/1000/60/60);
	diff-= hour*1000*60*60;
	var min= Math.floor(diff/1000/60);
	diff-= min*1000*60;
	var sec=Math.floor(diff/1000);
	var maxh = el.getAttribute('hour');
	var maxm = el.getAttribute('min');
	var maxs = el.getAttribute('sec');
	if((hour>maxh) || (hour==maxh && min>maxm) || (hour==maxh && min==maxm && sec>maxs)) //If we passed the time
		return "Finished";
	return "Now at: "+hour+":"+min+":"+sec+"\\"+maxh+":"+maxm+":"+maxs;
}

var syncwatch = (function(){
	var el = document.querySelector('syncwatch');
	return (el!=null);
})();


(function mouikkai(){
	if (syncwatch){
		setTimeout(function(){
				$('syncwatch').each(function(){	
					$(this).text(timer_from_el(this));
				});
				mouikkai();
			} ,1000);
	}else{
		setTimeout(function(){mouikkai()} ,10000); //if there are no syncwatches we search for one every 10 secons
	}
})();