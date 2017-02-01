update threads
	set log = array_append(log, $2::bytea)
	where id = $1
	returning pg_notify('t:' || $1, '')
