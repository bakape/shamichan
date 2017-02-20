select t.board, t.postCtr, t.imageCtr, t.replyTime, t.bumpTime, t.subject,
		(select array_length(t.log, 1)) as logCtr,
		p.editing, p.banned, p.spoiler, t.id, p.time, p.body, p.name, p.trip,
		p.auth, p.links, p.backlinks, p.commands, p.imageName, i.*
	from threads as t
	inner join posts as p
		on t.id = p.id
			and (p.deleted is null or p.deleted = 'false')
	left outer join images as i
		on p.SHA1 = i.SHA1
	order by bumpTime desc
