delete from bans
	where board = $1 and forPost = $2
	returning
		pg_notify('bans_upated', ''),
		log_moderation(1::smallint, $1, $2, $3)
