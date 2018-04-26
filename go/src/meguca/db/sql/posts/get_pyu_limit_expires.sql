select expires
	from pyu_limit
	where ip = $1 and board = $2
