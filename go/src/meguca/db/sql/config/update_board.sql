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
		rbText = $10,
		defaultCSS = $11,
		title = $12,
		notice = $13,
		rules = $14,
		eightball = $15
	where id = $1
	returning pg_notify('board_updated', $1)
