update boards
	set
		readOnly = $2,
		textOnly = $3,
		forcedAnon = $4,
		disableRobots = $5,
		title = $6,
		notice = $7,
		rules = $8,
		eightball = $9
	where id = $1
	returning pg_notify('board_updated', $1)
