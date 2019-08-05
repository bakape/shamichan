select
	id,
	read_only,
	text_only,
	forced_anon,
	disable_robots,
	flags,
	nsfw,
	rb_text,
	pyu,
	default_css,
	title,
	notice,
	rules,
	eightball
from boards
where id = $1
