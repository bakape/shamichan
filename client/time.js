/*
Timezone corrections, batch timestamp updates, syncwatch, etc.
 */

let main = require('./main'),
	{$, Backbone, common, oneeSama, options, state} = main;

function date_from_time_el(el) {
	if (!el)
		return new Date();
	const dTime = el.getAttribute('datetime');
	// Don't crash the function, if scanning an unsynced post in progress
	if (!dTime)
		return new Date();
	return new Date(dTime
		.replace(/-/g, '/')
		.replace('T', ' ')
		.replace('Z', ' GMT')
	);
}
main.reply('time:fromEl', date_from_time_el);

// Get a more accurate server-client time offset, for interclient syncing
// Does not account for latency, but good enough for our purposes
var serverTimeOffset = 0;
main.dispatcher[common.GET_TIME] = function(msg){
	if (!msg[0])
		return;
	serverTimeOffset = msg[0] - new Date().getTime();
};
main.reply('time:offset', serverTimeOffset);

let renderTimer;
function batcTimeRender(model, rtime = options.get('relativeTime')) {
	let models = state.posts.models;
	for (let i = 0, l = models.length; i < l; i++) {
		models[i].dispatch('renderTime')
	}
	if (renderTimer)
		clearTimeout(renderTimer);
	if (rtime)
		renderTimer = setTimeout(batcTimeRender, 60000)
}
main.comply('time:render', batcTimeRender);
options.on('change:relativeTime', batcTimeRender);

const is_skewed = (function(){
	var el = document.querySelector('time');
	if (!el)
		return false;
	var d = date_from_time_el(el);
	return oneeSama.readableTime(d.getTime()) != el.innerHTML;
})();

if (is_skewed) {
	// Rerender all post times. If relative time is enabled, the timestamps
	// will be rerender anyway in a minute, so no need for this.
	if (!oneeSama.rTime)
		batcTimeRender();

	setTimeout(function () {
		// next request, have the server render the right times
		$.cookie('timezone', -new Date().getTimezoneOffset() / 60, {
			expires: 90,
			path: '/'
		});
	}, 3000);
}

/* syncwatch */
function timer_from_el($el) {
	if (!serverTimeOffset)
		return;
	$el.addClass('timerTicking');
	const start= $el.attr('start'),
		end = $el.attr('end'),
		maxh = common.pad($el.attr('hour')),
		maxm = common.pad($el.attr('min')),
		maxs = common.pad($el.attr('sec'));

	(function moumouikkai(){
		// Prevent memory leak
		if (!$el.length)
			return;
		const now = common.serverTime();
		if (now > end)
			return $el.text('Finished');
		// If the start time is in the future
		if (start > now) {
			var countdown = Math.round((start - now) / 1000);
			if(countdown == 10 || countdown == 5)
				main.command('time:syncwatch', countdown);
			$el.text('Countdown: ' + countdown);
			return setTimeout(moumouikkai, 1000);
		}
		var diff = now - start,
			hour = Math.floor(diff / 1000 /60 / 60);
		diff -= hour * 1000 * 60 * 60;
		var min = Math.floor( diff / 1000 / 60);
		diff -= min * 1000 * 60;
		var sec = Math.floor(diff / 1000);
		$el.text("Now at: " + common.pad(hour) + ":" + common.pad(min) + ":"
			+ common.pad(sec) + " / " + maxh + ":" + maxm + ":" + maxs);
		return setTimeout(moumouikkai, 1000);
	})();
}

function mouikkai() {
	setTimeout(function() {
		main.$threads.find('syncwatch').not('.timerTicking').each(function() {
			timer_from_el($(this));
		});
		mouikkai();
	}, 1000);
}

main.defer(mouikkai)
	.defer(function() {
		// Append UTC clock to the top of the schedule
		let seconds;
		let el = document.getElementById('UTCClock').firstChild;
		el.addEventListener('click', handler);

		function handler() {
			seconds = true;
			this.removeAttribute('title');
			this.style.cursor = 'default';
			this.removeEventListener('click', handler);
			render();
		}

		function render() {
			if (!serverTimeOffset)
				return setTimeout(render, 1000);
			let d = new Date(common.serverTime()),
				html = oneeSama.readableTime(d);
			if (seconds)
				html += ':' + common.pad(d.getUTCSeconds());
			html += ' UTC';
			el.innerHTML = html;
			setTimeout(render, seconds ? 1000 : 60000);
		}

		render();
	});
