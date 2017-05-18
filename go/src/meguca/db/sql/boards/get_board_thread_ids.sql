select id from threads
	where board = $1
	order by
		sticky desc,
		bumpTime desc
