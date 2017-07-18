update boards
	set
		readOnly = $2,
		textOnly = $3,
		forcedAnon = $4,
		disableRobots = $5,
		flags = $6,
		NSFW = $7,
		defaultCSS = $8,
		title = $9,
		notice = $10,
		rules = $11,
		eightball = $12
	where id = $1
	returning pg_notify('board_updated', $1)
