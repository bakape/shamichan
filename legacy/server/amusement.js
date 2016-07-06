function parseSyncwatch(frag) {
	// First capture group may or may not be present
	const sw = frag.match(/^#sw(\d+:)?(\d+):(\d+)([+-]\d+)?$/i)
	if (!sw)
		return false
	const hour = parseInt(sw[1], 10) || 0,
		min = parseInt(sw[2], 10),
		sec = parseInt(sw[3], 10)
	let start = common.serverTime()

	// Offset the start. If the start is in the future, a countdown will be
	// displayed.
	if (sw[4]) {
		const symbol = sw[4].slice(0, 1),
			offset = sw[4].slice(1) * 1000
		start = symbol == '+' ? start + offset : start - offset
	}
	const end = ((hour * 60 + min) * 60 + sec) * 1000 + start

	return [common.tupleTypes.syncwatch, sec, min, hour, start, end]
}
