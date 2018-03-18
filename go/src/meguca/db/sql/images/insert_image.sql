update posts
set SHA1 = $2,
	imageName = $3,
	spoiler = $4
where id = $1
returning bump_thread(op, false, false, true)
