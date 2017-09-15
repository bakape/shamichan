update posts
	set spoiler = true
	where id = $1
	returning log_moderation(4::smallint, board, $1, $2),
		bump_thread(op, false, false, false)
