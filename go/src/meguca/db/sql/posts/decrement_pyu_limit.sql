update pyu_limit
	set pcount = pcount - 1
	where ip = $1 and board = $2
