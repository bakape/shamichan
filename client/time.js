/*
Timezone corrections, batch timestamp updates, syncwatch, etc.
 */

let main = require('./main'),
	{$, Backbone, common, oneeSama, options, state} = main;

// Get a more accurate server-client time offset, for interclient syncing
// Does not account for latency, but good enough for our purposes
var serverTimeOffset = 0;
main.dispatcher[common.GET_TIME] = function(msg){
	if (!msg[0])
		return;
	serverTimeOffset = msg[0] - Date.now();
};
main.reply('time:offset', serverTimeOffset);

let renderTimer;
function batcTimeRender(source, rtime = options.get('relativeTime')) {
	state.posts.each(model => model.dispatch('renderTime'));
	if (renderTimer)
		clearTimeout(renderTimer);
	if (rtime)
		renderTimer = setTimeout(batcTimeRender, 60000)
}
main.reply('time:render', batcTimeRender);
options.on('change:relativeTime', batcTimeRender);

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
				main.request('time:syncwatch', countdown);
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

main.defer(batcTimeRender)
	.defer(mouikkai)
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
			el.innerHTML = oneeSama
				.readableUTCTime(new Date(common.serverTime()), seconds);
			setTimeout(render, seconds ? 1000 : 60000);
		}

		render();
	});
