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
main.reply('time:offset', () => serverTimeOffset);

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
function timer_from_el(el) {
	if (!serverTimeOffset)
		return;
	el.classList.add('timerTicking');
	const start = el.getAttribute('start'),
		end = el.getAttribute('end'),
		maxh = common.pad(el.getAttribute('hour')),
		maxm = common.pad(el.getAttribute('min')),
		maxs = common.pad(el.getAttribute('sec'));

	(function moumouikkai() {
		// Prevent memory leak
		if (!document.body.contains(el))
			return;
		const now = common.serverTime();
		if (now > end)
			return el.textContent = main.lang.finished;

		// If the start time is in the future
		if (start > now) {
			const countdown = Math.round((start - now) / 1000);
			if(countdown === 10)
				main.request('time:syncwatch');
			el.textContent = 'Countdown: ' + countdown;
			return setTimeout(moumouikkai, 1000);
		}

		let diff = now - start;
		const hour = Math.floor(diff / 1000 /60 / 60);
		diff -= hour * 1000 * 60 * 60;
		const min = Math.floor( diff / 1000 / 60);
		diff -= min * 1000 * 60;
		const sec = Math.floor(diff / 1000);
		el.textContent = common.pad(hour) + ":" + common.pad(min) + ":"
			+ common.pad(sec) + " / " + maxh + ":" + maxm + ":" + maxs;
		return setTimeout(moumouikkai, 1000);
	})();
}

function mouikkai() {
	setInterval(function() {
		const els = document.getElementsByTagName('syncwatch');
		for (let i = 0; i < els.length; i++) {
			if (els[i].classList.contains('timerTicking'))
				continue;
			timer_from_el(els[i]);
		}
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
