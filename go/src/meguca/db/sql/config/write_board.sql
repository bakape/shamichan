insert into boards (
	id, readOnly, textOnly, forcedAnon, disableRobots, created, title,	notice,
	rules, eightball
)
	values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	returning pg_notify('board_updated', $1)
