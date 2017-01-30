select t.board, t.id, t.postCtr, t.imageCtr, t.replyTime, t.bumpTime, t.subject,
		p.spoiler, p.time, p.name, p.trip, p.auth, p.imageName,
		(select array_length(t.log, 1)) as logCtr, i.*
	from threads as t
	inner join posts as p
		on t.id = p.id and (deleted is null or deleted = 'false')
	left outer join images as i
		on p.SHA1 = i.SHA1
	where t.board = $1
	order by bumpTime desc
