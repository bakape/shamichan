update posts
	set banned = true
	where id = $1
	returning bump_thread(op, false, false, false)
