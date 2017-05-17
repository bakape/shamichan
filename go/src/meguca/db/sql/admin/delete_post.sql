update posts
	set deleted = true
	where id = $1
	returning log_moderation(2::smallint, board, id, $2::varchar(20))
