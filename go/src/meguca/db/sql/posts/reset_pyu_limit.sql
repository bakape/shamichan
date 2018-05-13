update pyu_limit
	set pcount = 4
	where ip = $1 and board = $2
	returning pcount
