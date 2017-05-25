update threads
	set sticky = $2
	where id = $1
	returning bump_thread($1, false, false, false)
