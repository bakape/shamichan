select id, time, editing, (SHA1 is not null) as hasImg from posts
	where op = $1
		and time > floor(extract(epoch from now())) - 900
	order by id asc
