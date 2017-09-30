insert into boards (
	id, readOnly, textOnly, forcedAnon, disableRobots, flags, NSFW, nonLive,
	posterIDs,
	created, defaultCSS, title,	notice, rules, eightball, js
)
	values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	returning pg_notify('board_updated', $1)
