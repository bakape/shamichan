select exists (
	select
	from bans
	where ip = $1
		and board in ('all', $2)
		and expires > now()
)
