class OneeSama {
	readableUTCTime(d, seconds) {
		let html = pad(d.getUTCDate()) + ' '
			+ this.lang.year[d.getUTCMonth()] + ' '
			+ d.getUTCFullYear()
			+ `(${this.lang.week[d.getUTCDay()]})`
			+`${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
		if (seconds)
			html += `:${pad(d.getUTCSeconds())}`;
		html += ' UTC';
		return html;
	}
}
