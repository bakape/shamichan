select t.sticky, t.board, t.postCtr, t.imageCtr, t.replyTime, t.bumpTime,
		t.subject, t.nonLive, t.locked,
		p.editing, p.banned, p.spoiler, p.deleted, p.sage, t.id, p.time, p.body,
		p.flag, p.name, p.trip, p.auth, p.links, p.commands, p.imageName,
		posterID,
		i.*
	from threads as t
	inner join posts as p
		on t.id = p.id
	left outer join images as i
		on p.SHA1 = i.SHA1
	order by bumpTime desc
