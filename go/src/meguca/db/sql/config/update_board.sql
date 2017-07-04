update boards
	set
		readOnly = $2,
		textOnly = $3,
		forcedAnon = $4,
		disableRobots = $5,
		flags = $6,
		defaultCSS = $7,
		title = $8,
		notice = $9,
		rules = $10,
		eightball = $11
	where id = $1
	returning pg_notify('board_updated', $1)
