update boards
set
	read_only = $2,
	text_only = $3,
	forced_anon = $4,
	disable_robots = $5,
	flags = $6,
	nsfw = $7,
	rb_text = $8,
	pyu = $9,
	created = $10
	default_css = $11,
	title = $12,
	notice = $13,
	rules = $14,
	eightball = $15
where id = $1
