update threads
	set log = log || $2::bytea[]
	where id = $1
	returning pg_notify('t:' || $1, '')
