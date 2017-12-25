update boards
	set
		readOnly = $2,
		textOnly = $3,
		forcedAnon = $4,
		disableRobots = $5,
		flags = $6,
		NSFW = $7,
		posterIDs = $8,
		defaultCSS = $9,
		title = $10,
		notice = $11,
		rules = $12,
		eightball = $13
	where id = $1
	returning pg_notify('board_updated', $1)
