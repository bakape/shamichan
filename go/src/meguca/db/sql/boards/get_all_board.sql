select t.sticky, t.board, t.postCtr, t.imageCtr, t.replyTime, t.bumpTime,
		t.subject,
		p.editing, p.banned, p.spoiler, p.deleted, p.sage, t.id, p.time, p.body,
		p.name, p.trip, p.auth, p.links, p.backlinks, p.commands, p.imageName,
		i.*
	from threads as t
	inner join posts as p
		on t.id = p.id
	left outer join images as i
		on p.SHA1 = i.SHA1
	order by bumpTime desc
