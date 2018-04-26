update pyu_limit
	set expires = $1
	where ip = $2 and board = $3
