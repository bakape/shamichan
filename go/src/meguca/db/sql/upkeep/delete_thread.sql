delete from threads
	where id = $1
	returning pg_notify('thread_deleted', board || ':' || id)
