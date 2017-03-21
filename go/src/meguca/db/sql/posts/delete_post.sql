update posts
	set deleted = true
	where id = $1
	returning update_log(op, $2::bytea);
