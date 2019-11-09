select exists (
	select
	from sessions
	where account = $1
		and token = $2
		and expires < now()
)
