main.reply('time:syncwatch', function () {
	if (!options.get('notification') || !document.hidden)
		return;
	new Notification(main.lang.syncwatchStarting)
		.onclick = () => window.focus();
});
