update threads
	set locked = $2
	where id = $1
	returning log_moderation(5::smallint, board, $1, $3),
		bump_thread($1, false, false, false)
