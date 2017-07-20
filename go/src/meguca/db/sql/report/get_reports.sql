select id, target, reason, created
	from reports
	where board = $1
	order by created desc
