select position from staff
	where board = $1 and account = $2
	limit 1
