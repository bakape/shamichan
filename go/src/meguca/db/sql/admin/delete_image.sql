update posts
	set SHA1 = null
	where id = $1
	returning log_moderation(3::smallint, board, id, $2::varchar(20)),
		bump_thread(op, false, false, false)
