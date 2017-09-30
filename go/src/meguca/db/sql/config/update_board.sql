update boards
	set
		readOnly = $2,
		textOnly = $3,
		forcedAnon = $4,
		disableRobots = $5,
		flags = $6,
		NSFW = $7,
		nonLive = $8,
		posterIDs = $9,
		defaultCSS = $10,
		title = $11,
		notice = $12,
		rules = $13,
		eightball = $14,
		js = $15
	where id = $1
	returning pg_notify('board_updated', $1)
