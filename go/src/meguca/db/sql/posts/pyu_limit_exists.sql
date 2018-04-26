select count(1)
	from pyu_limit
	where ip = $1 and board = $2
